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
