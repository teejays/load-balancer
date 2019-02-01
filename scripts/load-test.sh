#!/bin/bash

for i in `seq 1 100000`; 
do
    curl http://localhost:8888 &
    sleep 0.001
done