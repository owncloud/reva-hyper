## oCIS Wrapper

A tool that wraps the oCIS binary and provides a way to re-configure the running oCIS instance.

When run, **ociswrapper** starts an API server that exposes some endpoints to re-configure the oCIS server.

### Usage

1.  Build

    ```bash
    make build
    ```

2.  Run

    ```bash
    ./bin/ociswrapper serve --bin=<path-to-ocis-binary>
    ```

    To check other available options:

    ```bash
    ./bin/ociswrapper serve --help
    ```

    ```bash
     --url string              oCIS server url (default "https://localhost:9200")
     --retry string            Number of retries to start oCIS server (default "5")
     -p, --port string         Wrapper API server port (default "5200")
     --admin-username string   admin username for oCIS server
     --admin-password string   admin password for oCIS server
    ```

Access the API server at `http://localhost:5200`.

Also, see `./bin/ociswrapper help` for more information.

### API

**ociswrapper** exposes two endpoints:

1.  `PUT /config`

    Updates the configuration of the running oCIS instance.
    Body of the request should be a JSON object with the following structure:

    ```json
    {
      "ENV_KEY1": "value1",
      "ENV_KEY2": "value2"
    }
    ```

    Returns:

    - `200 OK` - oCIS is successfully reconfigured
    - `400 Bad Request` - request body is not a valid JSON object
    - `500 Internal Server Error` - oCIS server is not running

2.  `DELETE /rollback`

    Rolls back the configuration to the starting point.

    Returns:

    - `200 OK` - rollback is successful
    - `500 Internal Server Error` - oCIS server is not running

3.  `POST /command`

    Executes the provided command on the oCIS server. The body of the request should be a JSON object with the following structure:

    ```yml
    {
      "command": "<ocis-command>", # without the ocis binary. e.g. "list"
    }
    ```

    If the command requires user input, the body of the request should be a JSON object with the following structure:

    ```json
    {
      "command": "<ocis-command>",
      "inputs": ["value1"]
    }
    ```

    Returns:

    ```json
    {
      "status": "OK",
      "exitCode": 0,
      "message": "<command output>"
    }
    OR
    {
      "status": "ERROR",
      "exitCode": <error-exit-code>,
      "message": "<command output>"
    }
    ```

    - `200 OK` - command is successfully executed
    - `400 Bad Request` - request body is not a valid JSON object
    - `500 Internal Server Error`

4.  `POST /start`

    Starts the oCIS server.

    Returns:

    - `200 OK` - oCIS server is started
    - `409 Conflict` - oCIS server is already running
    - `500 Internal Server Error` - Unable to start oCIS server

5.  `POST /stop`

    Stops the oCIS server.

    Returns:

    - `200 OK` - oCIS server is stopped
    - `500 Internal Server Error` - Unable to stop oCIS server

6. `POST /services/{service-name}`

    Restart oCIS with service excluded and start excluded oCIS service individually, not covered by the oCIS supervisor.

    Body of the request should be a JSON object with the following structure:

    ```json
    {
      "ENV_KEY1": "value1",
      "ENV_KEY2": "value2"
    }
    ```

    Returns:

    - `200 OK` - oCIS server is stopped
    - `500 Internal Server Error` - Unable to stop oCIS server

7. `DELETE /services/{service-name}`

   Stop individually running oCIS service

   Returns:

    - `200 OK` - command is successfully executed
    - `400 Bad Request` - request body is not a valid JSON object
    - `500 Internal Server Error`
