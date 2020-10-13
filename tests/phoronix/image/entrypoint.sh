#!/bin/bash
echo "Do batch-setup"
printf "n\nn\n" | /usr/bin/phoronix-test-suite batch-setup
#echo "Disable DynamicRunCount"
#sed -i "s/<DynamicRunCount>TRUE<\/DynamicRunCount>/<DynamicRunCount>FALSE<\/DynamicRunCount>/" /etc/phoronix-test-suite.xml
echo "Modify LimitDynamicToTestLength"
sed -i "s/<LimitDynamicToTestLength>20<\/LimitDynamicToTestLength>/<LimitDynamicToTestLength>5<\/LimitDynamicToTestLength>/" /etc/phoronix-test-suite.xml
echo "Modify EnvironmentDirectory"
sed -i "s/<EnvironmentDirectory>~\/.phoronix-test-suite\/installed-tests\/<\/EnvironmentDirectory>/<EnvironmentDirectory>\/data\/<\/EnvironmentDirectory>/" /etc/phoronix-test-suite.xml
echo "Run benchmark"
/usr/bin/phoronix-test-suite batch-benchmark "$BENCHMARK" | tee /var/www/html/index.html
echo "Serve results via http"
nginx -g "daemon off;"
