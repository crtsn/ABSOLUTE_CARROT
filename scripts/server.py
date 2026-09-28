#!/usr/bin/env python3
import http.server

PORT = 8080

from http.server import SimpleHTTPRequestHandler

class MyHandler(SimpleHTTPRequestHandler):
    def send_error(self, code, message=None):
        if code == 404:
            with open("404.html") as html_file:
                self.error_message_format = html_file.read().replace('%', '%%')
        SimpleHTTPRequestHandler.send_error(self, code, message)

if __name__ == "__main__":
    try:
        with http.server.ThreadingHTTPServer(("", PORT), MyHandler) as httpd:
            print(f"Serving at http://127.0.0.1:{PORT}/ (Ctrl+C to stop)")
            httpd.serve_forever()
    except KeyboardInterrupt:
        print("Keyboard interrupt detected")
