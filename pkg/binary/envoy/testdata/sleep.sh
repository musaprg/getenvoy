#!/bin/bash

# Copyright 2019 Tetrate
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script is used to simulate a process that never terminates unless it receives SIGINT or SIGTERM
trap 'echo >&2 "SIGTERM caught!"; exit 0' TERM
trap 'echo >&2 "SIGINT caught!"; exit 0' INT

while true; do
	sleep 1
	if [ "$1" == "error" ]; then
		exit 1
	fi
done
