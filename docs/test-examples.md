# Test examples

Just some examples of running tests:

```bash
# EDEN setup
make build-tests
eden setup
source ~/.eden/activate.sh
eden+ports.sh 5912:5902 5911:5901 8027:8027 8028:8028 2223:2223 2224:2224
eden start
eden onboard

## tests/app
eden test tests/app -p eden.escript.test -r TestEdenScripts/2dockers_test
eden eve reset
## tests/docker
eden test tests/docker -p eden.escript.test -r TestEdenScripts/2dockers_test
eden eve reset
## tests/eclient
# Just a simple test of eclient image functionality -- tested in more complex tests
#eden test tests/eclient -p eden.escript.test -r TestEdenScripts/eclient -t 15m
eden test tests/eclient -p eden.escript.test -r TestEdenScripts/host-only -t 15m
eden eve reset
eden test tests/eclient -p eden.escript.test -r TestEdenScripts/networking_light -t 15m
eden eve reset
eden test tests/eclient -p eden.escript.test -r TestEdenScripts/nw_switch -t 20m
eden eve reset
eden test tests/eclient -p eden.escript.test -r TestEdenScripts/port_switch -t 20m
eden eve reset
# Just a simple test of nginx image -- tested in port_switch test
#eden test tests/eclient -p eden.escript.test -r TestEdenScripts/ngnix -t 20m
eden test tests/eclient -p eden.escript.test -r TestEdenScripts/maridb -t 20m
eden eve reset
## tests/escript
eden test tests/escript -p eden.escript.test -r TestEdenScripts/arg -a "-args=test1=123,test2=456"
eden eve reset
eden test tests/escript -p eden.escript.test -r TestEdenScripts/template
eden eve reset
eden test tests/escript -p eden.escript.test -r TestEdenScripts/message
eden eve reset
eden test tests/escript -p eden.escript.test -r TestEdenScripts/nested_scripts
eden eve reset
eden test tests/escript -p eden.escript.test -r TestEdenScripts/time
eden eve reset
eden test tests/escript -p eden.escript.test -r TestEdenScripts/source
eden eve reset
eden test tests/escript -p eden.escript.test -r TestEdenScripts/fail_scenario
eden eve reset
## tests/lim
eden test tests/lim -p eden.escript.test -r TestEdenScripts/log_test
eden eve reset
eden test tests/lim -p eden.escript.test -r TestEdenScripts/info_test
eden eve reset
eden test tests/lim -p eden.escript.test -r TestEdenScripts/metric_test
eden eve reset
## tests/network
eden test tests/network -p eden.escript.test -r TestEdenScripts/test_networking -t 40m
eden eve reset
## tests/phoronix
eden test tests/phoronix -p eden.escript.test -r TestEdenScripts/test_phoronix -a "benchmark=fio-basic"
eden eve reset
## tests/reboot
eden test tests/reboot -p eden.reboot.test
eden eve reset
## tests/units
make -C tests/units test
## tests/update_eve_image
eden test tests/update_eve_image -p eden.escript.test -r TestEdenScripts/update_eve_image -t 10m
eden eve reset
## tests/vnc
eden test tests/vnc -p eden.vnc.test
## tests/workflow
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/log_test -a '-testdata ../lim/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/ssh
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/info_test -a '-testdata ../lim/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/metric_test -a '-testdata ../lim/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/test_networking -a '-testdata ../network/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/2dockers_test -a '-testdata ../app/testdata/'
eden eve reset
eden test tests/workflow -p eden.vnc.test
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/host-only -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/networking_light -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/ngnix -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/maridb -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/reboot_test
eden eve reset
eden test tests/workflow -p eden.escript.test -r TestEdenScripts/update_eve_image -a '-testdata ../update_eve_image/testdata/'
```
