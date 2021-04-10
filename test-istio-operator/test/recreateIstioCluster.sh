#!/bin/bash

set -ex

kubectl apply -f istio-1.yaml
sleep 70s
kubectl delete Istio sonar-istio-cluster
sleep 60s
kubectl apply -f istio-1.yaml
sleep 80s
