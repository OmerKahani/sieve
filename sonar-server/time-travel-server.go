package main

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	sonar "sonar.client"
)

var globalCntToRestart int = 0
var mutex = &sync.Mutex{}

// The listener is actually a wrapper around the server.
func NewTimeTravelListener(config map[interface{}]interface{}) *TimeTravelListener {
	server := &timeTravelServer{
		project:     config["project"].(string),
		seenPrev:    false,
		paused:      false,
		restarted:   false,
		pauseCh:     make(chan int),
		straggler:   config["straggler"].(string),
		crucialCur:  config["ce-diff-current"].(string),
		crucialPrev: config["ce-diff-previous"].(string),
		podLable:    config["operator-pod"].(string),
		frontRunner: config["front-runner"].(string),
		command:     config["command"].(string),
	}
	listener := &TimeTravelListener{
		Server: server,
	}
	listener.Server.Start()
	return listener
}

type TimeTravelListener struct {
	Server *timeTravelServer
}

// Echo is just for testing.
func (l *TimeTravelListener) Echo(request *sonar.EchoRequest, response *sonar.Response) error {
	*response = sonar.Response{Message: "echo " + request.Text, Ok: true}
	return nil
}

func (l *TimeTravelListener) NotifyTimeTravelCrucialEvent(request *sonar.NotifyTimeTravelCrucialEventRequest, response *sonar.Response) error {
	return l.Server.NotifyTimeTravelCrucialEvent(request, response)
}

func (l *TimeTravelListener) NotifyTimeTravelRestartPoint(request *sonar.NotifyTimeTravelRestartPointRequest, response *sonar.Response) error {
	return l.Server.NotifyTimeTravelRestartPoint(request, response)
}

func (l *TimeTravelListener) NotifyTimeTravelSideEffects(request *sonar.NotifyTimeTravelSideEffectsRequest, response *sonar.Response) error {
	return l.Server.NotifyTimeTravelSideEffects(request, response)
}

type timeTravelServer struct {
	project     string
	straggler   string
	frontRunner string
	crucialCur  string
	crucialPrev string
	podLable    string
	seenPrev    bool
	paused      bool
	restarted   bool
	pauseCh     chan int
	command     string
}

func (s *timeTravelServer) Start() {
	log.Println("start timeTravelServer...")
}

func strToMap(str string) map[string]interface{} {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		log.Fatalf("cannot unmarshal to map: %s\n", str)
	}
	return m
}

func (s *timeTravelServer) equivalentEventList(crucialEvent, currentEvent []interface{}) bool {
	if len(crucialEvent) != len(currentEvent) {
		return false
	}
	for i, val := range crucialEvent {
		switch v := val.(type) {
		case int64:
			if e, ok := currentEvent[i].(int64); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case float64:
			if e, ok := currentEvent[i].(float64); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case bool:
			if e, ok := currentEvent[i].(bool); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case string:
			if v == "SONAR-NON-NIL" || v == "SONAR-SKIP" {
				continue
			} else if e, ok := currentEvent[i].(string); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case map[string]interface{}:
			if e, ok := currentEvent[i].(map[string]interface{}); ok {
				if !s.equivalentEvent(v, e) {
					return false
				}
			} else {
				return false
			}
		default:
			log.Printf("Unsupported type: %v %T\n", v, v)
			return false
		}
	}
	return true
}

func (s *timeTravelServer) equivalentEvent(crucialEvent, currentEvent map[string]interface{}) bool {
	for key, val := range crucialEvent {
		if _, ok := currentEvent[key]; !ok {
			return false
		}
		switch v := val.(type) {
		case int64:
			if e, ok := currentEvent[key].(int64); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case float64:
			if e, ok := currentEvent[key].(float64); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case bool:
			if e, ok := currentEvent[key].(bool); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case string:
			if v == "SONAR-NON-NIL" {
				continue
			} else if e, ok := currentEvent[key].(string); ok {
				if v != e {
					return false
				}
			} else {
				return false
			}
		case map[string]interface{}:
			if e, ok := currentEvent[key].(map[string]interface{}); ok {
				if !s.equivalentEvent(v, e) {
					return false
				}
			} else {
				return false
			}
		case []interface{}:
			if e, ok := currentEvent[key].([]interface{}); ok {
				if !s.equivalentEventList(v, e) {
					return false
				}
			} else {
				return false
			}
		default:
			log.Printf("Unsupported type: %v %T\n", v, v)
			return false
		}
	}
	return true
}

func (s *timeTravelServer) equivalentEventSecondTry(crucialEvent, currentEvent map[string]interface{}) bool {
	if _, ok := currentEvent["metadata"]; ok {
		return false
	}
	if metadataMap, ok := crucialEvent["metadata"]; ok {
		if m, ok := metadataMap.(map[string]interface{}); ok {
			for key := range m {
				crucialEvent[key] = m[key]
			}
			delete(crucialEvent, "metadata")
			return s.equivalentEvent(crucialEvent, currentEvent)
		} else {
			return false
		}
	} else {
		return false
	}
}

func (s *timeTravelServer) isCrucial(crucialEvent, currentEvent map[string]interface{}) bool {
	if s.equivalentEvent(crucialEvent, currentEvent) {
		log.Println("Meet")
		return true
	} else if s.equivalentEventSecondTry(crucialEvent, currentEvent) {
		log.Println("Meet for the second try")
		return true
	} else {
		return false
	}
}

func (s *timeTravelServer) NotifyTimeTravelCrucialEvent(request *sonar.NotifyTimeTravelCrucialEventRequest, response *sonar.Response) error {
	log.Printf("NotifyTimeTravelCrucialEvent: Hostname: %s\n", request.Hostname)
	if s.straggler != request.Hostname {
		*response = sonar.Response{Message: request.Hostname, Ok: true}
		return nil
	}
	currentEvent := strToMap(request.Object)
	crucialCurEvent := strToMap(s.crucialCur)
	crucialPrevEvent := strToMap(s.crucialPrev)
	log.Printf("[sonar][current-event] %s\n", request.Object)
	if s.shouldPause(crucialCurEvent, crucialPrevEvent, currentEvent) {
		log.Println("[sonar] should sleep here")
		<-s.pauseCh
		log.Println("[sonar] sleep over")
	}
	*response = sonar.Response{Message: request.Hostname, Ok: true}
	return nil
}

func (s *timeTravelServer) NotifyTimeTravelRestartPoint(request *sonar.NotifyTimeTravelRestartPointRequest, response *sonar.Response) error {
	log.Printf("NotifyTimeTravelSideEffect: Hostname: %s\n", request.Hostname)
	if s.frontRunner != request.Hostname {
		*response = sonar.Response{Message: request.Hostname, Ok: true}
		return nil
	}
	log.Printf("[sonar][restart-point] %s %s %s %s\n", request.Name, request.Namespace, request.ResourceType, request.EventType)
	if s.shouldRestart() {
		log.Println("[sonar] should restart here")
		go s.waitAndRestartComponent()
	}
	*response = sonar.Response{Message: request.Hostname, Ok: true}
	return nil
}

func (s *timeTravelServer) NotifyTimeTravelSideEffects(request *sonar.NotifyTimeTravelSideEffectsRequest, response *sonar.Response) error {
	name, namespace := extractNameNamespace(request.Object)
	log.Printf("[SONAR-SIDE-EFFECT]\t%s\t%s\t%s\t%s\t%s\n", request.SideEffectType, request.ResourceType, namespace, name, request.Error)
	*response = sonar.Response{Message: request.SideEffectType, Ok: true}
	return nil
}

func (s *timeTravelServer) waitAndRestartComponent() {
	time.Sleep(time.Duration(10) * time.Second)
	s.restartComponent(s.project, s.podLable)
	time.Sleep(time.Duration(20) * time.Second)
	s.pauseCh <- 0
}

func (s *timeTravelServer) shouldPause(crucialCurEvent, crucialPrevEvent, currentEvent map[string]interface{}) bool {
	if !s.paused {
		if !s.seenPrev {
			if s.isCrucial(crucialPrevEvent, currentEvent) {
				log.Println("Meet crucialPrevEvent: set seenPrev to true")
				s.seenPrev = true
			}
		} else {
			if s.isCrucial(crucialCurEvent, currentEvent) {
				log.Println("Meet crucialCurEvent: set paused to true and start to pause")
				s.paused = true
				return true
			}
			// else if s.isCrucial(crucialPrevEvent, currentEvent) {
			// 	log.Println("Meet crucialPrevEvent: keep seenPrev as true")
			// 	// s.seenPrev = true
			// } else {
			// 	log.Println("Not meet anything: set seenPrev back to false")
			// 	s.seenPrev = false
			// }
		}
	}
	return false
}

func (s *timeTravelServer) shouldRestart() bool {
	if s.paused && !s.restarted {
		s.restarted = true
		return true
	} else {
		return false
	}
}

// The controller to restart is identified by `operator-pod` in the configuration.
// `operator-pod` is a label to identify the pod where the controller is running.
// We do not directly use pod name because the pod belongs to a deployment so its name is not fixed.
func (s *timeTravelServer) restartComponent(project, podLabel string) {
	config, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	checkError(err)
	clientset, err := kubernetes.NewForConfig(config)
	checkError(err)
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"sonartag": podLabel}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), listOptions)
	checkError(err)
	if len(pods.Items) == 0 {
		log.Fatalln("didn't get any pod")
	}
	pod := pods.Items[0]
	log.Printf("get operator pod: %s\n", pod.Name)

	// The way we crash and restart the controller is not very graceful here.
	// The util.sh is a simple script with commands to kill the controller process
	// and restart the controller process.
	// Why not directly call the commands?
	// The command needs nested quotation marks and
	// I find parsing nested quotation marks are tricky in golang.
	// TODO: figure out how to make nested quotation marks work
	cmd := exec.Command("./util.sh", s.command, pod.Name, s.straggler)
	err = cmd.Run()
	checkError(err)
	log.Println("restart successfully")

	// cmd2 := exec.Command("./util.sh", s.command, pod.Name)
	// err = cmd2.Run()
	// checkError(err)
	// log.Println("restart")
}
