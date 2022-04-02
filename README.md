[![License: Apache 2](https://img.shields.io/badge/License-apache2-green.svg)](LICENSE)
[![mloperator](https://github.com/polyaxon/mloperator/actions/workflows/tests.yml/badge.svg)](https://github.com/polyaxon/mloperator/actions/workflows/tests.yml)
[![Slack](https://img.shields.io/badge/chat-on%20slack-aadada.svg?logo=slack&longCache=true)](https://polyaxon.com/slack/)
[![Docs](https://img.shields.io/badge/docs-stable-brightgreen.svg?style=flat)](https://polyaxon.com/docs/)
[![GitHub](https://img.shields.io/badge/issue_tracker-github-blue?logo=github)](https://github.com/polyaxon/polyaxon/issues)
[![GitHub](https://img.shields.io/badge/roadmap-github-blue?logo=github)](https://github.com/polyaxon/polyaxon/milestones)

<a href="https://polyaxon.com"><img src="https://raw.githubusercontent.com/polyaxon/polyaxon/master/artifacts/packages/mloperator.svg" width="125" height="125" align="right" /></a>

# Machine Learning Operator & Controller for Kubernetes

## Introduction

Kubernetes offers the facility of extending it's API through the concept of 'Operators' ([Introducing Operators: Putting Operational Knowledge into Software](https://coreos.com/blog/introducing-operators.html)). This repository contains the resources and code to deploy an Polyaxon native CRDs using a native Operator for Kubernetes.

This project is a Kubernetes controller that manages and watches Customer Resource Definitions (CRDs) that define primitives to handle, operate and reconcile operations like: builds, jobs, experiments, distributed training, notebooks, tensorboards, kubeflow integrations, ...

![MLOperator Architecture](./artifacts/MLOperator-architecture.png)

## Kubeflow operators

This Operator extends natively [Kubeflow-Operators](https://github.com/polyaxon/training-operator) (TFJob/PytorchJob/MXNet/XGBoost/MPI).
