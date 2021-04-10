#!/bin/bash

set -ex

kubectl apply -f istio-3.yaml
sleep 300
kubectl apply -f istio-2.yaml
sleep 180
