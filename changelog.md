All notable changes to this project will be documented in this file.

The versioning format follows [Semantic Versioning](https://semver.org/):**Major.Minor.Patch** (e.g., v2.3.5)

## [Unreleased]

- Work in progress

## [v1.0.0] - 2025-01-01

### Added

- **Initial Release:**
  - URL shortening service that generates unique short codes.
  - Redirection from short codes to original URLs.
  - Deletion of short code.

## [v1.2.1] - 2025-01-01

### Fixed

- Updated error message and HTTP status code for empty URL in shorten end point

### Added

- Added last_accessed and hit_count for each short code.

### Changed

- Multiple short code for same url is allowed

## [v2.0.0] - 2025-01-01

### Added

- **Breaking Change:** Added API key verification for shorten and delete of api
- Added expiry_at and custom_code for shorten api endpoint
- Added shorten-bulk endpoint for shortening more than one URL

## [v3.0.0] - 2025-01-01

### Added

- Introduced tier enterprise and hobby for shorten-bulk
- **Breaking Change:** The shorten-bulk endpoint is now restricted to users with the `enterprise` tier. Users with a `hobby` tier (or no tier) will no longer be able to use the bulk shortening functionality.
- Added edit request on shorten endpoint with expiry_at value
- Added an optional password field for shorten, edit URL's and redirect
- Added health endpoint
- Added logger for audit, debug and error
