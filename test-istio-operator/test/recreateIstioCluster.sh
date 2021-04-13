#!/bin/bash

set -ex

# kubectl apply -f istio-1.yaml
# sleep 150s
# kubectl delete Istio sonar-istio-cluster
# sleep 150s
# kubectl apply -f istio-1.yaml
# sleep 150s

kubectl apply -f remoteistio.yaml
sleep 60s
kubectl delete RemoteIstio sonar-remoteistio-cluster
sleep 60s
kubectl apply -f remoteistio.yaml
sleep 60s

