---
title: "Acceptance Testing"
date: 2018-05-02T00:00:00+00:00
weight: 20
geekdocRepo: https://github.com/owncloud/ocis
geekdocEditPath: edit/master/docs/ocis/development
geekdocFilePath: testing.md
---

{{< toc >}}

To run tests in the test suite you have two options. You may go the easy way and just run the test suite in docker. But for some tasks you could also need to install the test suite natively, which requires a little more setup since PHP and some dependencies need to be installed.

Both ways to run tests with the test suites are described here.

## Running Test Suite in Docker

Let's see what is available. Invoke the following command from within the root of the oCIS repository.

```bash
make -C tests/acceptance/docker help
```

Basically we have two sources for feature tests and test suites:

- [oCIS feature test and test suites](https://github.com/owncloud/ocis/tree/master/tests/acceptance/features)
- [tests and test suites transferred from ownCloud core, they have prefix coreApi](https://github.com/owncloud/ocis/tree/master/tests/acceptance/features)

At the moment, both can be applied to oCIS since the api of oCIS is designed to be compatible with ownCloud.

As a storage backend, we offer oCIS native storage, also called `ocis`. This stores files directly on disk. Along with that we also provide `S3` storage driver.

You can invoke two types of test suite runs:

- run a full test suite, which consists of multiple feature tests
- run a single feature or single scenario in a feature

### Run Full Test Suite

#### Local oCIS Tests (prefix `api`)

The names of the full test suite make targets have the same naming as in the CI pipeline. See the available local oCIS specific test suites [here](https://github.com/owncloud/ocis/tree/master/tests/acceptance/features). They can be run with `ocis` storage and `S3` storage.

For example, command:

```bash
make -C tests/acceptance/docker localApiTests-apiGraph-ocis
```

runs the same tests as the `localApiTests-apiGraph-ocis` CI pipeline, which runs the oCIS test suite "apiGraph" against the oCIS server with ocis storage.

And command:

```bash
make -C tests/acceptance/docker localApiTests-apiGraph-s3ng
```

runs the oCIS test suite `apiGraph` against the oCIS server with s3 storage.

{{< hint info >}}
While running the tests, oCIS server is started with [ociswrapper](https://github.com/owncloud/ocis/blob/master/tests/ociswrapper/README.md) (i.e. `WITH_WRAPPER=true`) by default. In order to run the tests without ociswrapper, provide `WITH_WRAPPER=false` when running the tests. For example:

```bash
WITH_WRAPPER=false \
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature:26' \
make -C tests/acceptance/docker test-ocis-feature-ocis-storage
```

But some test suites that are tagged with `@env-config` require the oCIS server to be run with ociswrapper. So, running those tests require `WITH_WRAPPER=true` (default setting).
{{< /hint >}}

{{< hint info >}}
To run the tests that require an email server (tests tagged with `@email`), you need to provide `START_EMAIL=true` while running the tests.

```bash
START_EMAIL=true \
BEHAT_FEATURE='tests/acceptance/features/apiNotification/emailNotification.feature' \
make -C tests/acceptance/docker test-ocis-feature-ocis-storage
```

{{< /hint >}}

{{< hint info >}}
To run the tests that require tika service (tests tagged with `@tikaServiceNeeded`), you need to provide `START_TIKA=true` while running the tests.

```bash
START_TIKA=true \
BEHAT_FEATURE='tests/acceptance/features/apiSearchContent/contentSearch.feature' \
make -C tests/acceptance/docker test-ocis-feature-ocis-storage
```

{{< /hint >}}

{{< hint info >}}
To run the tests that require an antivirus service (tests tagged with `@antivirus`), you need to provide the following environment variables while running the tests.

```bash
START_ANTIVIRUS=true \
OCIS_ASYNC_UPLOADS=true \
OCIS_ADD_RUN_SERVICES=antivirus \
POSTPROCESSING_STEPS=virusscan \
BEHAT_FEATURE='tests/acceptance/features/apiAntivirus/antivirus.feature' \
make -C tests/acceptance/docker test-ocis-feature-ocis-storage
```

{{< /hint >}}

#### Tests Transferred From ownCloud Core (prefix `coreApi`)

Command `make -C tests/acceptance/docker Core-API-Tests-ocis-storage-3` runs the same tests as the `Core-API-Tests-ocis-storage-3` CI pipeline, which runs the third (out of ten) test suite groups transferred from ownCloud core against the oCIS server with ocis storage.

And `make -C tests/acceptance/docker Core-API-Tests-s3ng-storage-3` runs the third (out of ten) test suite groups transferred from ownCloud core against the oCIS server with s3 storage.

### Run Single Feature Test

The tests for a single feature (a feature file) can also be run against the different storage backends. To do that, multiple make targets with the schema **test-_\<test-source\>_-feature-_\<storage-backend\>_** are available. To select a single feature you have to add an additional `BEHAT_FEATURE=<path-to-feature-file>` parameter when invoking the make command.

For example;

```bash
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature' \
make -C tests/acceptance/docker test-ocis-feature-ocis-storage
```

{{< hint info >}}
`BEHAT_FEATURE` must be pointing to a valid feature file
{{< /hint >}}

And to run a single scenario in a feature, you can do:

{{< hint info >}}
A specific scenario from a feature can be run by adding `:<line-number>` at the end of the feature file path. For example, to run the scenario at line 26 of the feature file `apiGraphUserGroup/createUser.feature`, simply add the line number like this: `apiGraphUserGroup/createUser.feature:26`. Note that the line numbers mentioned in the examples might not always point to a scenario, so always check the line numbers before running the test.
{{< /hint >}}

```bash
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature:26' \
make -C tests/acceptance/docker test-ocis-feature-ocis-storage
```

Similarly, with S3 storage;

```bash
# run a whole feature
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature' \
make -C tests/acceptance/docker test-ocis-feature-s3ng-storage

# run a single scenario
BEHAT_FEATURE='tests/acceptance/features/apiGraphUserGroup/createUser.feature:26' \
make -C tests/acceptance/docker test-ocis-feature-s3ng-storage
```

In the same way, tests transferred from ownCloud core can be run as:

```bash
# run a whole feature
BEHAT_FEATURE='tests/acceptance/features/coreApiAuth/webDavAuth.feature' \
make -C tests/acceptance/docker test-core-feature-ocis-storage

# run a single scenario
BEHAT_FEATURE='tests/acceptance/features/coreApiAuth/webDavAuth.feature:15' \
make -C tests/acceptance/docker test-core-feature-ocis-storage
```

{{< hint info >}}
The test suites transferred from ownCloud core have `coreApi` prefixed
{{< /hint >}}

### oCIS Image to Be Tested (Skip Local Image Build)

By default, the tests will be run against the docker image built from your current working state of the oCIS repository. For some purposes it might also be handy to use an oCIS image from Docker Hub. Therefore, you can provide the optional flag `OCIS_IMAGE_TAG=...` which must contain an available docker tag of the [owncloud/ocis registry on Docker Hub](https://hub.docker.com/r/owncloud/ocis) (e.g. 'latest').

```bash
OCIS_IMAGE_TAG=latest \
make -C tests/acceptance/docker localApiTests-apiGraph-ocis
```

### Test Log Output

While a test is running or when it is finished, you can attach to the logs generated by the tests.

```bash
make -C tests/acceptance/docker show-test-logs
```

{{< hint info >}}
The log output is opened in `less`. You can navigate up and down with your cursors. By pressing "F" you can follow the latest line of the output.
{{< /hint >}}

### Cleanup

During testing we start a redis and oCIS docker container. These will not be stopped automatically. You can stop them with:

```bash
make -C tests/acceptance/docker clean
```

## Running Test Suite in Local Environment

### Run oCIS

Create an up-to-date oCIS binary by [building oCIS]({{< ref "build" >}})

To start oCIS:

```bash
IDM_ADMIN_PASSWORD=admin \
ocis/bin/ocis init --insecure true

OCIS_INSECURE=true PROXY_ENABLE_BASIC_AUTH=true \
ocis/bin/ocis server
```

`PROXY_ENABLE_BASIC_AUTH` will allow the acceptance tests to make requests against the provisioning api (and other endpoints) using basic auth.

#### Run Local oCIS Tests (prefix `api`) and Tests Transferred From ownCloud Core (prefix `coreApi`)

```bash
make test-acceptance-api \
TEST_SERVER_URL=https://localhost:9200 \
```

Useful environment variables:

`TEST_SERVER_URL`: oCIS server url. Please, adjust the server url according to your setup.

`BEHAT_FEATURE`: to run a single feature

{{< hint info >}}
A specific scenario from a feature can be run by adding `:<line-number>` at the end of the feature file path. For example, to run the scenario at line 26 of the feature file `apiGraphUserGroup/createUser.feature`, simply add the line number like this: `apiGraphUserGroup/createUser.feature:26`. Note that the line numbers mentioned in the examples might not always point to a scenario, so always check the line numbers before running the test.
{{< /hint >}}

> Example:
>
> BEHAT_FEATURE=tests/acceptance/features/apiGraphUserGroup/createUser.feature
>
> Or
>
> BEHAT_FEATURE=tests/acceptance/features/apiGraphUserGroup/createUser.feature:13

`BEHAT_SUITE`: to run a single suite

> Example:
>
> BEHAT_SUITE=apiGraph

`STORAGE_DRIVER`: to run tests with a different user storage driver. Available options are `ocis` (default), `owncloudsql` and `s3ng`

> Example:
>
> STORAGE_DRIVER=owncloudsql

`STOP_ON_FAILURE`: to stop running tests after the first failure

> Example:
>
> STOP_ON_FAILURE=true

### Use Existing Tests for BDD

As a lot of scenarios are written for oC10, we can use those tests for Behaviour driven development in oCIS.
Every scenario that does not work in oCIS with "ocis" storage, is listed in `tests/acceptance/expected-failures-API-on-OCIS-storage.md` with a link to the related issue.

Those scenarios are run in the ordinary acceptance test pipeline in CI. The scenarios that fail are checked against the
expected failures. If there are any differences then the CI pipeline fails.

The tests are not currently run in CI with the OWNCLOUD or EOS storage drivers, so there are no expected-failures files for those.

If you want to work on a specific issue

1. locally run each of the tests marked with that issue in the expected failures file.

   E.g.:

   ```bash
   make test-acceptance-api \
   TEST_SERVER_URL=https://localhost:9200 \
   STORAGE_DRIVER=OCIS \
   BEHAT_FEATURE='tests/acceptance/features/coreApiVersions/fileVersions.feature:141'
   ```

2. the tests will fail, try to understand how and why they are failing
3. fix the code
4. go back to 1. and repeat till the tests are passing.
5. remove those tests from the expected failures file
6. make a PR that has the fixed code, and the relevant lines removed from the expected failures file.

## Running Tests With And Without `remote.php`

By default, the tests are run with `remote.php` enabled. If you want to run the tests without `remote.php`, you can disable it by setting the environment variable `WITH_REMOTE_PHP=false` while running the tests.

```bash
WITH_REMOTE_PHP=false \
TEST_SERVER_URL="https://localhost:9200" \
make test-acceptance-api
```

## Running ENV Config Tests (@env-Config)

Test suites tagged with `@env-config` are used to test the environment variables that are used to configure oCIS. These tests are special tests that require the oCIS server to be run using [ociswrapper](https://github.com/owncloud/ocis/blob/master/tests/ociswrapper/README.md).

### Run oCIS With ociswrapper

```bash
# working dir: ocis repo root dir

# init oCIS
IDM_ADMIN_PASSWORD=admin \
ocis/bin/ocis init --insecure true

# build the wrapper
cd tests/ociswrapper
make build

# run oCIS
PROXY_ENABLE_BASIC_AUTH=true \
./bin/ociswrapper serve --bin=../../ocis/bin/ocis
```

### Run the Tests

```bash
OCIS_WRAPPER_URL=http://localhost:5200 \
TEST_SERVER_URL="https://localhost:9200" \
BEHAT_FEATURE=tests/acceptance/features/apiAsyncUpload/delayPostprocessing.feature \
make test-acceptance-api
```

### Writing New ENV Config Tests

While writing tests for a new oCIS ENV configuration, please make sure to follow these guidelines:

1. Tag the test suite (or test scenarios) with `@env-config`
2. Use `OcisConfigHelper.php` for helper functions - provides functions to reconfigure the running oCIS instance.
3. Recommended: add the new step implementations in `OcisConfigContext.php`

## Running Test Suite With Email Service (@email)

Test suites that are tagged with `@email` require an email service. We use inbucket as the email service in our tests.

### Setup Inbucket

Run the following command to setup inbucket

```bash
docker run -d -p9000:9000 -p2500:2500 --name inbucket inbucket/inbucket
```

### Run oCIS

Documentation for environment variables is available [here](https://owncloud.dev/services/notifications/#environment-variables)

```bash
# init oCIS
IDM_ADMIN_PASSWORD=admin \
ocis/bin/ocis init --insecure true

# run oCIS
PROXY_ENABLE_BASIC_AUTH=true \
OCIS_ADD_RUN_SERVICES=notifications \
NOTIFICATIONS_SMTP_HOST=localhost \
NOTIFICATIONS_SMTP_PORT=2500 \
NOTIFICATIONS_SMTP_INSECURE=true \
NOTIFICATIONS_SMTP_SENDER="owncloud <noreply@example.com>" \
ocis/bin/ocis server
```

### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
EMAIL_HOST="localhost" \
EMAIL_PORT=9000 \
BEHAT_FEATURE="tests/acceptance/features/apiNotification/emailNotification.feature" \
make test-acceptance-api
```

## Running Test Suite With Tika Service (@tikaServiceNeeded)

Test suites that are tagged with `@tikaServiceNeeded` require tika service.

### Setup Tika Service

Run the following docker command to setup tika service

```bash
docker run -d -p 127.0.0.1:9998:9998 apache/tika
```

### Run oCIS

Documentation related to the content based search and tika extractor can be found [here](https://doc.owncloud.com/ocis/next/deployment/services/s-list/search.html#content-extraction)

```bash
# init oCIS
IDM_ADMIN_PASSWORD=admin \
ocis/bin/ocis init --insecure true

# run oCIS
PROXY_ENABLE_BASIC_AUTH=true \
OCIS_INSECURE=true \
SEARCH_EXTRACTOR_TYPE=tika \
SEARCH_EXTRACTOR_TIKA_TIKA_URL=http://localhost:9998 \
SEARCH_EXTRACTOR_CS3SOURCE_INSECURE=true \
ocis/bin/ocis server
```

### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
BEHAT_FEATURE="tests/acceptance/features/apiSearchContent/contentSearch.feature" \
make test-acceptance-api
```

## Running Test Suite With Antivirus Service (@antivirus)

Test suites that are tagged with `@antivirus` require antivirus service. The available antivirus and the configuration related to them can be found [here](https://doc.owncloud.com/ocis/next/deployment/services/s-list/antivirus.html). This documentation is only going to use `clamAv` as antivirus.

### Setup clamAV

#### 1. Setup Locally

##### Linux OS user

Run the following command to set up calmAV and clamAV daemon

```bash
sudo apt install clamav clamav-daemon -y
```

Make sure that the clamAV daemon is up and running

```bash
sudo service clamav-daemon status
```

{{< hint info >}}
The commands are ubuntu specific and may differ according to your system. You can find information related to installation of clamAV in their official documentation [here](https://docs.clamav.net/manual/Installing/Packages.html).
{{< /hint>}}

##### Mac OS user

Install ClamAV using [here](https://gist.github.com/mendozao/3ea393b91f23a813650baab9964425b9)
Start ClamAV daemon

```bash
/your/location/to/brew/Cellar/clamav/1.1.0/sbin/clamd
```

#### 2. Setup clamAV With Docker

##### Linux OS user

Run `clamAV` through docker

```bash
docker run -d -p 3310:3310 owncloudci/clamavd
```

##### Mac OS user

```bash
docker run -d -p 3310:3310 -v /your/local/filesystem/path/to/clamav/:/var/lib/clamav mkodockx/docker-clamav:alpine
```

### Run oCIS

As `antivirus` service is not enabled by default we need to enable the service while running oCIS server. We also need to enable `async upload` and as virus scan is performed in post-processing step, we need to set it as well. Documentation for environment variables related to antivirus is available [here](https://owncloud.dev/services/antivirus/#environment-variables)

```bash
# init oCIS
IDM_ADMIN_PASSWORD=admin \
ocis/bin/ocis init --insecure true

# run oCIS
PROXY_ENABLE_BASIC_AUTH=true \
ANTIVIRUS_SCANNER_TYPE="clamav" \
ANTIVIRUS_CLAMAV_SOCKET="tcp://host.docker.internal:3310" \
POSTPROCESSING_STEPS="virusscan" \
OCIS_ASYNC_UPLOADS=true \
OCIS_ADD_RUN_SERVICES="antivirus"
ocis/bin/ocis server
```

{{< hint info >}}
The value for `ANTIVIRUS_CLAMAV_SOCKET` is an example which needs adaption according your OS.

For antivirus running localy on Linux OS, use `ANTIVIRUS_CLAMAV_SOCKET= "/var/run/clamav/clamd.ctl"`.
For antivirus running localy on Mac OS, use `ANTIVIRUS_CLAMAV_SOCKET= "/tmp/clamd.socket"`.
For antivirus running with docker, use `ANTIVIRUS_CLAMAV_SOCKET= "tcp://host.docker.internal:3310"`
{{< /hint>}}

#### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
BEHAT_FEATURE="tests/acceptance/features/apiAntivirus/antivirus.feature" \
make test-acceptance-api
```

## Running Test Suite With Federated Sharing (@ocm)

Test suites that are tagged with `@ocm` require running two different ocis instances. More detailed information and configuration related to it can be found [here](https://doc.owncloud.com/ocis/5.0/deployment/services/s-list/ocm.html).

### Setup First oCIS Instance

```bash
# init oCIS
IDM_ADMIN_PASSWORD=admin \
ocis/bin/ocis init --insecure true

# run oCIS
OCIS_URL="https://localhost:9200" \
PROXY_ENABLE_BASIC_AUTH=true \
OCIS_ENABLE_OCM=true \
OCM_OCM_PROVIDER_AUTHORIZER_PROVIDERS_FILE="tests/config/local/providers.json" \
OCIS_ADD_RUN_SERVICES="ocm" \
OCM_OCM_INVITE_MANAGER_INSECURE=true \
OCM_OCM_SHARE_PROVIDER_INSECURE=true \
OCM_OCM_STORAGE_PROVIDER_INSECURE=true \
WEB_UI_CONFIG_FILE="tests/config/local/ocis-web.json" \
ocis/bin/ocis server
```

The first oCIS instance should be available at: https://localhost:9200/

### Setup Second oCIS Instance

You can run the second oCIS instance in two ways:

#### Using `.vscode/launch.json`

From the `Run and Debug` panel of VSCode, select `Fed oCIS Server` and start the debugger.

#### Using env file

```bash
# init oCIS
source tests/config/local/.env-federation && ocis/bin/ocis init

# run oCIS
ocis/bin/ocis server
```

The second oCIS instance should be available at: https://localhost:10200/

{{< hint info >}}
To enable ocm in the web interface, you need to set the following envs:
`OCIS_ENABLE_OCM="true"`
`OCIS_ADD_RUN_SERVICES="ocm"`
{{< /hint>}}

#### Run the Acceptance Test

Run the acceptance test with the following command:

```bash
TEST_SERVER_URL="https://localhost:9200" \
TEST_SERVER_FED_URL="https://localhost:10200" \
BEHAT_FEATURE="tests/acceptance/features/apiOcm/ocm.feature" \
make test-acceptance-api
```

## Running Text Preview Tests Containing Unicode Characters

There are some tests that check the text preview of files containing Unicode characters. The oCIS server by default cannot generate the thumbnail of such files correctly but it provides an environment variable to allow the use of custom fonts that support Unicode characters. So to run such tests successfully, we have to run the oCIS server with this environment variable.

```bash
...
THUMBNAILS_TXT_FONTMAP_FILE="/path/to/fontsMap.json"
ocis/bin/ocis server
```

The sample `fontsMap.json` file is located in `tests/config/drone/fontsMap.json`.

```json
{
  "defaultFont": "/path/to/ocis/tests/config/drone/NotoSans.ttf"
}
```


## Running Test Suite With Document Servers  (Collabora, ONLYOFFICE or Microsoft using the WOPI protocol.) with dokcer
To run the test related to document Servers, go to `tests/acceptance/docker/documentServer` and run the command
```bash
   docker compose up
```
Latest ocis build is done with local ocis docker image that is build with this docker compose file.
oCIS will start in `https://ocis.owncloud.test/` along with all other service.
