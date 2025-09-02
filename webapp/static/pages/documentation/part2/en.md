---
title: 'API Reference: v2 Endpoints'
description: A comprehensive list of all RESTful API endpoints available in version 2, including request and response schemas.
date: 2025-03-20
tags:
  - documentation
  - api
  - reference
updatedOn: 2025-05-05
---

Below is the list of **v2 API endpoints**. Each endpoint includes the HTTP method, URL path, required parameters, and example responses.

### 1. Authentication

**POST** `/api/v2/auth/login`

- **Description**: Authenticate a user and return a JWT token.
- **Request Body**:
  ```json
  {
  	"email": "user@example.com",
  	"password": "Password123!"
  }
  ```
