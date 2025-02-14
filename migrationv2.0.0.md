# Migration Guide: Upgrading to v2.0.0

## Introduction

Welcome to the API 1.2.1 to 2.0.0 Migration Guide. This document provides comprehensive information and guidelines for migrating your applications from API version 1.2.0 to version 2.0.0 . The new version introduces several enhancements, optimisations, and changes to improve overall performance and functionality.

## Overview

Version 2.0.0 brings several important updates:

- **API Key Verification:** All requests to the shorten and delete endpoints now require a valid API key.
- **New Fields for the Shorten Endpoint:** The API now accepts `expiry_at` (expiration timestamp) and `custom_code` (a custom alias for the shortened URL) as optional fields.
- **Bulk URL Shortening:** A new `/shorten-bulk` endpoint has been introduced to allow shortening multiple URLs in one request.

This guide will walk you through the necessary changes to update your integration.

## Breaking Changes

1. **API Key Requirement:**
   - **Before:** The API might not have required an API key.
   - **Now:** Every request to the shorten and delete endpoints must include a valid API key in the header.
2. **Updated Shorten Endpoint:**
   - **New Fields:**
     - `expiry_at`: (Optional) A timestamp indicating when the short URL should expire.
     - `custom_code`: (Optional) A custom alias for the shortened URL.
   - If these fields are not provided, the endpoint behaves as before for backward compatibility.
3. **Bulk Shortening Endpoint:**
   - A new endpoint, `/shorten-bulk`, is available for shortening multiple URLs in a single request.

## Migration Steps

### 1. Update Your Request Headers for API Key Verification

- **Action:**

  - Retrieve your API key from the developer portal.
  - Include it in your request headers:

    **Example:**

    ```http
    api_key: your-api-key
    ```

    **In Go:**

    ```go
    req.Header.Set("api_key", "your-api-key")
    ```

### 2. Modify the Shorten Endpoint Payload (if needed)

- **New Fields:**  
  Add `expiry_at` and/or `custom_code` to your JSON payload when calling the shorten endpoint.

  **Example Payload:**

  ```json
  {
    "long_url": "https://www.example.com",
    "expiry_at": "2025-12-31T23:59:59Z",
    "custom_code": "myCustomAlias"
  }
  ```
