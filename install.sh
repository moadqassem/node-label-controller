#!/usr/bin/env bash
kubectl apply -f deploy/controller-roles.yaml
kubectl apply -f deploy/configmap.yaml
kubectl apply -f deploy/labelling-controller.yaml
kubectl apply -f deploy/container-linux-update-operator.yaml