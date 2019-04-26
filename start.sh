#!/bin/bash

docker run -p 9000:9000 -e TWILIO_SID="$1" -e TWILIO_TOKEN="$2" -e FROM="$3" -e CALLBACK="$4" --restart unless-stopped ancientlore/faxxr
