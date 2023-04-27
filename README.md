# Enlabs Test Application

This application provides an API for processing transactions and managing user balances.

## Requirements

- go
- Docker
- Docker Compose
- Make
- Bash
- Git
- Curl
- Postman (optional)

## How to run the application

1. Clone the repository: git clone `https://github.com/mufasadev/enlabs-test.git`
2. Go to the project directory: cd enlabs-test
3. Run go mod download to download the dependencies and go mod vendor to create a vendor directory
4. Edit .env. Setup LOCAL_PORT (make sure it is not used by another application) 
5. Setup DB_PORT (make sure it is not used by another application)
6. Run make install wait for the docker containers to be built and started
7. To rebuild the containers run make up_build. To rebuild the containers and drop the database run make up_build_all


## Testing the application with curl

To test the application manually using `curl`, you can send POST requests to the `/transactions` endpoint and GET requests to the `/balance` endpoint.

### Sending a transaction

To send a transaction, use the following `curl` command:

```bash
curl -X POST -H "Content-Type: application/json" -H "Source-Type: game" -d '{"state": "win", "amount": "510.15", "transactionId": "some_uuid"}' http://localhost:8080/api/v1/users/f60ae2e1-ee72-4a6a-bef2-7cde5c83782f/transactions
```
Replace game with either server or payment as needed, and replace some_uuid with a unique identifier for the transaction.

### Getting the user balance
To get the user balance, use the following curl command:
    
```bash
curl http://localhost:8080/api/v1/users/f60ae2e1-ee72-4a6a-bef2-7cde5c83782f/balance
```
This command will return the current balance of the user as a JSON object.

## API Endpoints

The application provides the following endpoints:

`POST /api/v1/users/{userId}/transactions`

For the test application, the userId is always `f60ae2e1-ee72-4a6a-bef2-7cde5c83782f`

This endpoint processes a transaction for a user.

#### Headers:

- `Content-Type: application/json`
- `Source-Type: game|server|payment`

#### Request body:

```json
{
  "state": "win|lost",
  "amount": "123.45",
  "transactionId": "some_uuid"
}
```
Response:
- 200 OK: The transaction was successfully processed.
- 400 Bad Request: The request body is invalid or missing required fields.
- 422 Unprocessable Entity: The transaction could not be processed.
- 500 Internal Server Error: An error occurred while processing the transaction.

This endpoint retrieves the current balance for a user. 

`GET /api/v1/users/{userId}/balance`

Response:
- 200 OK: The user's balance as a JSON object.
- 404 Not Found: The specified user was not found.
- 500 Internal Server Error: An error occurred while retrieving the balance.

## Running tests
To run the test script, you can use the provided small script, which simulates sending transactions and prints the user's balance at the end:

```bash
make test_requests
```
Make sure the application is running before running the script.

This script will send transactions for 30 seconds using 10 concurrent workers, and then print the final user balance.

## Running the application with Postman
To test the application using Postman, you can create a new collection and add the following requests:

Sending a transaction
1. Create a new POST request with the URL:
`http://localhost:8080/api/v1/users/f60ae2e1-ee72-4a6a-bef2-7cde5c83782f/transactions`
2. Add headers:
   - Content-Type: application/json
   - Source-Type: game (or server or payment)
3. Set the request body to raw and JSON format, then provide the JSON data:
    ```json   
       {
          "state": "win",
          "amount": "510.15",
          "transactionId": "some_uuid"
       }
    ```
4. Replace some_uuid with a unique identifier for the transaction.
5. Click "Send" to process the transaction.

## Getting the user balance
1. Create a new GET request with the URL: 
`http://localhost:8080/api/v1/users/f60ae2e1-ee72-4a6a-bef2-7cde5c83782f/balance`
2. Click "Send" to retrieve the user's balance. The response should be a JSON object containing the user's current balance.

## Running unit tests for database
To run the unit tests for the database, run the following command:
```bash
make test_unit
```
This command will run the unit tests inside the container and display the results.

Note: Make sure the application is running before executing the unit tests.

## Monitoring and Logs
To monitor the application's logs and check the status of the running containers, you can use the following Makefile commands:
```bash
make logs
```
This command will display real-time logs for all the services defined in the Docker Compose configuration.

```bash
make ps
```
This command will show the list of running containers and their current status.

## Troubleshooting
If you encounter any issues while running the application, you can try the following steps:

1. Ensure that the ports specified in the .env file are not being used by other applications.
2. Check the logs for any errors using the `make logs` command.
3. Restart the application using the `make up_build` command.
4. Rebuild the application using the `make up_build_all` command.

## Conclusion and Future Improvements

In apps with many users, one big query is often better than many small ones. This is 
because it reduces the back-and-forth between the app and the database, making things 
faster. Also, the database can work more efficiently with one query.

There was some confusion in the task, so two query versions were created and tested. 
The first query doesn't create a transaction if the user's balance update fails, 
while the second query creates a transaction in any case. However, if the user's 
balance update is unsuccessful, the transaction status will be marked as unprocessed. 
Tests were done for both, but only the second one is used in the app. This shows that 
the solution is flexible and can be changed if needed.

The third query is responsible for canceling transactions. It cancels only a part of 
the transactions that can be reversed, and if the user's balance doesn't allow some 
transactions to be undone, they are ignored.

It's important to note that the code may still require refactoring, as the primary 
focus was on creating a working solution. Less time was spent on architecture, which 
means that there might be a room for improvement in the future.