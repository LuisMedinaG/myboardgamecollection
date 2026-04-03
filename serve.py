#!/usr/bin/env python3
"""Local dev server. Zero dependencies — uses Python's built-in http.server."""

import http.server
import sys

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8000

handler = http.server.SimpleHTTPRequestHandler
with http.server.HTTPServer(("127.0.0.1", PORT), handler) as server:
    print(f"Serving at http://127.0.0.1:{PORT}")
    server.serve_forever()
