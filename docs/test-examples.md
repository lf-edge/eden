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
eden test tests/app -e 2dockers_test
eden eve reset
## tests/docker
eden test tests/docker -e 2dockers_test
eden eve reset
## tests/eclient
# Just a simple test of eclient image functionality -- tested in more complex tests
#eden test tests/eclient -e eclient -t 15m
eden test tests/eclient -e host-only -t 15m
eden eve reset
eden test tests/eclient -e networking_light -t 15m
eden eve reset
eden test tests/eclient -e nw_switch -t 20m
eden eve reset
eden test tests/eclient -e port_switch -t 20m
eden eve reset
# Just a simple test of nginx image -- tested in port_switch test
#eden test tests/eclient -e nginx -t 20m
eden test tests/eclient -e maridb -t 20m
eden eve reset
## tests/escript
eden test tests/escript -e arg -a "-args=test1=123,test2=456"
eden eve reset
eden test tests/escript -e template
eden eve reset
eden test tests/escript -e message
eden eve reset
eden test tests/escript -e nested_scripts
eden eve reset
eden test tests/escript -e time
eden eve reset
eden test tests/escript -e source
eden eve reset
eden test tests/escript -e fail_scenario
eden eve reset
## tests/lim
eden test tests/lim -e log_test
eden eve reset
eden test tests/lim -e info_test
eden eve reset
eden test tests/lim -e metric_test
eden eve reset
## tests/network
eden test tests/network -e test_networking -t 40m
eden eve reset
## tests/phoronix
eden test tests/phoronix -e test_phoronix -a "benchmark=fio-basic"
eden eve reset
## tests/reboot
eden test tests/reboot -p eden.reboot.test
eden eve reset
## tests/units
make -C tests/units test
## tests/update_eve_image
eden test tests/update_eve_image -e update_eve_image_http -t 10m
eden eve reset
## tests/vnc
eden test tests/vnc -p eden.vnc.test
## tests/workflow
eden test tests/workflow -e log_test -a '-testdata ../lim/testdata/'
eden eve reset
eden test tests/workflow -e ssh
eden eve reset
eden test tests/workflow -e info_test -a '-testdata ../lim/testdata/'
eden eve reset
eden test tests/workflow -e metric_test -a '-testdata ../lim/testdata/'
eden eve reset
eden test tests/workflow -e test_networking -a '-testdata ../network/testdata/'
eden eve reset
eden test tests/workflow -e 2dockers_test -a '-testdata ../app/testdata/'
eden eve reset
eden test tests/workflow -p eden.vnc.test
eden eve reset
eden test tests/workflow -e host-only -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -e networking_light -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -e nginx -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -e maridb -a '-testdata ../eclient/testdata/'
eden eve reset
eden test tests/workflow -e reboot_test
eden eve reset
eden test tests/workflow -e update_eve_image_http -a '-testdata ../update_eve_image/testdata/'
```
