#!/bin/bash

set -ex

sleep 60s # wait for manager ready

kubectl apply -f remoteistio.yaml
sleep 60s
kubectl delete RemoteIstio sonar-remoteistio-cluster
sleep 60s
kubectl apply -f remoteistio.yaml
sleep 60s

