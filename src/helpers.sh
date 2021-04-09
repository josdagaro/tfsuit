#!/bin/bash

die()
{
    echo "$1"
    exit "${2:-0}"
}
