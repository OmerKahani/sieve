#!/bin/bash

set -ex

kubectl apply -f zkc-1.yaml
if [ $1 = 'learn' ]; then sleep 300s; else sleep 30s; fi
kubectl delete ZookeeperCluster zookeeper-cluster
if [ $1 = 'learn' ]; then sleep 300s; else sleep 25; fi
kubectl apply -f zkc-1.yaml
if [ $1 = 'learn' ]; then sleep 300s; else sleep 40s; fi

