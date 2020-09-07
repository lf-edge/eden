#!/bin/bash
echo "Do batch-setup"
printf "n\nn\n" | /usr/bin/phoronix-test-suite batch-setup
echo "Run benchmark"
/usr/bin/phoronix-test-suite batch-benchmark "$BENCHMARK" | tee /var/www/html/index.html
echo "Serve results via http"
nginx -g "daemon off;"
