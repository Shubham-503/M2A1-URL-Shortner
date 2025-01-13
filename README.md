# URL Shortener Service

### Installation and Setup

1. Clone the repository:

   Git clone:
   cd url_shortener

2. Install dependencies:

   go mod tidy

3. Run the server:

   go run main.go

4. The server will start at http://localhost:8080.

### Running Tests

1. Run the unit tests:

   go test -v

### Latency Report

/shorten

times=[4,4,4,5,5,5,5,6,6,7]

p50=5, p90=6.1, p95=6.6, p99=6.9

/redirect

times=[1,2,2,2,2,2,2,3,3,3]

p50=2, p90=3, p95=3, p99=3

## Schema Design

### URLShortener Table

| Column         | Type         | Description                                                                |
| -------------- | ------------ | -------------------------------------------------------------------------- |
| ID             | `uint`       | Primary key                                                                |
| OriginalURL    | `string`     | The original URL that was shortened                                        |
| ShortCode      | `string`     | Unique short code assigned to the URL                                      |
| ShortenCount   | `unit`       | Number of times the original URL has been shortened with the same API key. |
| HitCount       | `unit`       | Number of times the short code has been accessed                           |
| Password       | `*string`    | (Optional) Password protection for the short code                          |
| ApiKey         | `string`     | Associated API key for the user                                            |
| CreatedAt      | `*time.Time` | Created date for the short code                                            |
| ExpiredAt      | `*time.Time` | Expiry date for the short code                                             |
| LastAccessedAt | `*time.Time` | Timestamp of the last access                                               |
| DeletedAt      | `*time.Time` | Timestamp of when the URL was deleted (soft delete)                        |
| UserID         | `uint`       | Foreign key linking to the User table                                      |

---

### User Table

| Column    | Type        | Description                                           |
| --------- | ----------- | ----------------------------------------------------- |
| ID        | `uint`      | Primary key                                           |
| Email     | `string`    | User's email address                                  |
| Name      | `string`    | User's name                                           |
| ApiKey    | `string`    | Unique API key assigned to the user                   |
| Tier      | `string`    | Subscription tier of the user (`hobby`, `enterprise`) |
| CreatedAt | `time.Time` | Timestamp of when the user was created                |

---

## API Endpoints

### 1. **POST `/shorten`**

This endpoint generates a short code for a given long URL. It also supports optional features such as setting a custom short code, an expiration date, and password protection.

#### Request Headers

| **Header** | **Description**                              | **Required** |
| ---------- | -------------------------------------------- | ------------ |
| `api_key`  | The API key for the user making the request. | Yes          |

#### Request Body

The request body should be a JSON object with the following fields:

| **Field**     | **Type**           | **Description**                                                                                                     | **Required** |
| ------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------- | ------------ |
| `long_url`    | `string`           | The original long URL to shorten.                                                                                   | Yes          |
| `custom_code` | `string`           | An optional custom short code. If not provided, a random code will be generated.                                    | No           |
| `expired_at`  | `string` (ISO8601) | The optional expiration date and time for the short code. Must be in ISO8601 format (e.g., `2025-01-31T23:59:59Z`). | No           |
| `password`    | `string`           | An optional password to protect access to the short code.                                                           | No           |

#### Example Request

```json
{
  "long_url": "https://example.com",
  "custom_code": "example123",
  "expired_at": "2025-01-31T23:59:59Z",
  "password": "securepassword"
}
```

### 2. **POST `/shorten-bulk`**

This endpoint allows **enterprise users** to bulk shorten multiple URLs in a single API request. Each URL can have optional features such as a custom short code, expiration date, and password protection.

#### Request Headers

| **Header** | **Description**                                                                                                      | **Required** |
| ---------- | -------------------------------------------------------------------------------------------------------------------- | ------------ |
| `api_key`  | The API key for the user making the request. Bulk shortening is only available for users with the `enterprise` tier. | Yes          |

#### Request Body

The request body should be a JSON object containing an array of URLs, each with the following fields:

| **Field**     | **Type**           | **Description**                                                                                                     | **Required** |
| ------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------- | ------------ |
| `long_url`    | `string`           | The original long URL to shorten.                                                                                   | Yes          |
| `custom_code` | `string`           | An optional custom short code. If not provided, a random code will be generated.                                    | No           |
| `expired_at`  | `string` (ISO8601) | The optional expiration date and time for the short code. Must be in ISO8601 format (e.g., `2025-01-31T23:59:59Z`). | No           |
| `password`    | `string`           | An optional password to protect access to the short code.                                                           | No           |

#### Example Request

```json
{
  "urls": [
    {
      "long_url": "https://example.com/page1",
      "custom_code": "custom1",
      "expired_at": "2025-01-31T23:59:59Z",
      "password": "securepassword1"
    },
    {
      "long_url": "https://example.com/page2",
      "custom_code": "",
      "expired_at": null,
      "password": null
    }
  ]
}
```

### 3. **GET `/redirect`**

This endpoint allows users to retrieve the original URL associated with a given short code. If a password or expiration date is set for the short code, additional validation will be performed.

#### Request Parameters

The endpoint accepts the following query parameters:

| **Parameter** | **Type** | **Description**                                        | **Required**                           |
| ------------- | -------- | ------------------------------------------------------ | -------------------------------------- |
| `code`        | `string` | The short code associated with the original URL.       | Yes                                    |
| `password`    | `string` | The password to access the short code, if one was set. | No (unless required by the short code) |

#### Example Request

GET /redirect?code=abc123&password=securepassword

#### Response

The response contains the original URL associated with the short code. If the short code has a password, the user must provide it to successfully retrieve the URL.

| **Field**  | **Type** | **Description**                                                |
| ---------- | -------- | -------------------------------------------------------------- |
| `long_url` | `string` | The original long URL associated with the provided short code. |

#### Example Response

```json
{
  "long_url": "https://example.com/page1"
}
```

### 4. **DELETE `/redirect`**

This endpoint allows users to delete a short code associated with their API key by marking it as deleted in the database (`deleted_at` field).

#### Request Parameters

The endpoint accepts the following query parameters:

| **Parameter** | **Type** | **Description**               | **Required** |
| ------------- | -------- | ----------------------------- | ------------ |
| `code`        | `string` | The short code to be deleted. | Yes          |

#### Headers

| **Header** | **Type** | **Description**                       | **Required** |
| ---------- | -------- | ------------------------------------- | ------------ |
| `api_key`  | `string` | The API key associated with the user. | Yes          |

#### Example Request

DELETE /redirect?code=abc123 Header: api_key: your-api-key

#### Response

The response contains a message indicating whether the deletion was successful.

| **Field** | **Type** | **Description**                                  |
| --------- | -------- | ------------------------------------------------ |
| `message` | `string` | A success message if the short code was deleted. |

#### Example Response (Success)

```json
{
  "message": "short code deleted successfully"
}
```

### 5. **PATCH `/redirect`**

This endpoint allows users to update the `expired_at` and/or `password` fields of an existing short code. The request requires an `api_key` for authorization and the short code as a query parameter.

#### Request Parameters

The endpoint accepts the following query parameters:

| **Parameter** | **Type** | **Description**               | **Required** |
| ------------- | -------- | ----------------------------- | ------------ |
| `code`        | `string` | The short code to be updated. | Yes          |

#### Headers

| **Header** | **Type** | **Description**                       | **Required** |
| ---------- | -------- | ------------------------------------- | ------------ |
| `api_key`  | `string` | The API key associated with the user. | Yes          |

#### Request Body

The request body should be a JSON object containing one or both of the following fields:

| **Field**    | **Type**   | **Description**                             | **Required** |
| ------------ | ---------- | ------------------------------------------- | ------------ |
| `expired_at` | `datetime` | The new expiration date for the short code. | No           |
| `password`   | `string`   | The new password for the short code.        | No           |

#### Example Request

PATCH /redirect?code=abc123 Header: api_key: your-api-key

Body: { "expired_at": "2025-01-15T00:00:00Z", "password": "newpassword" }

#### Response

The response contains a message indicating whether the update was successful.

| **Field** | **Type** | **Description**                                 |
| --------- | -------- | ----------------------------------------------- |
| `message` | `string` | A success message if the update was successful. |

#### Example Response (Success)

```json
{
  "message": "Update Successful"
}
```

### 6. **GET `/users/url`**

This endpoint retrieves all URLs created by a user associated with the provided API key.

#### Headers

| **Header** | **Type** | **Description**                       | **Required** |
| ---------- | -------- | ------------------------------------- | ------------ |
| `api_key`  | `string` | The API key associated with the user. | Yes          |

#### Request Parameters

No query parameters or request body are required for this endpoint.

#### Example Request

GET /users/url Header: api_key: your-api-key

#### Response

The response is a JSON array of URLs created by the user, including details like the short code, original URL, expiration date, and hit count.

##### Response Format

```json
[
  {
    "id": 1,
    "short_code": "abc123",
    "original_url": "https://example.com",
    "expired_at": "2025-01-15T00:00:00Z",
    "hit_count": 10,
    "last_accessed_at": "2025-01-10T12:00:00Z",
    "created_at": "2025-01-01T10:00:00Z",
    "updated_at": "2025-01-10T12:00:00Z"
  },
  {
    "id": 2,
    "short_code": "xyz456",
    "original_url": "https://another-example.com",
    "expired_at": null,
    "hit_count": 5,
    "last_accessed_at": null,
    "created_at": "2025-01-05T08:00:00Z",
    "updated_at": "2025-01-06T14:00:00Z"
  }
]
```

### 7. **GET `/health`**

This endpoint checks the health of the server and the database connectivity. It returns the status of the system, indicating whether the server and database are functioning correctly.

#### Example Request

GET /health

#### Response

The response is a JSON object indicating the health status of the server and database.

##### Success Response (200 OK)

```json
{
  "status": "healthy",
  "message": "Server and database are up and running"
}
```
