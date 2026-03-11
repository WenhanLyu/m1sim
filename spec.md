# Project Specification

## What do you want to build?

M1Sim is a cycle-accurate, execution-driven Apple M1 CPU simulator built on the Akita framework.

## How do you consider the project is success?

We need to support simulating the whole polybench benchmark suite with 20% average time estimation error. The maximum error of each benchmark execution cannot be greater than 50%. Error is calculated by error = abs(sim-hw)/min(sim,hw).

You run on an Apple M1 Max machine, so you can run locally to measure.
