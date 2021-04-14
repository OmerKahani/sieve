#!/bin/bash

set -ex

sleep 60s # wait for manager ready

kubectl apply -f istio-3.yaml
sleep 300
kubectl apply -f istio-2.yaml
sleep 180
