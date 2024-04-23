# CCS Test Suite

## How to run test locally ...
### Installation
- Make sure you are running python 3.11 - python 3.12 has troubles with the code base
- git clone git@github.com:DeepDiver1975/ccs-caldavtester.git
- cd ccs-caldavtester
- python3 -m venv venv
- source venv/bin/activate
- pip install -r requirements.txt

### Run tests
- run in ./tests/ccs
- python3 ccs-caldavtester/testcaldav.py --ssl --print-details-onfail --basedir "." \
"CalDAV/caldavIOP.xml"

## Alternative
- docker pull deepdiver/ccs-caldavtester:latest
- see .drone.star
