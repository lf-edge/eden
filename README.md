# Eden

Eden is where [EVE](https://github.com/lf-edge/eve) and [Adam](https://github.com/lf-edge/adam) get tried and tested.

Eden consists of a test harness and a series of integration tests implemented in Golang. Tests are structured as normal Golang tests by using ```_test.go``` nomenclature and be available for test runs using standard go test framework.

## Install Dependencies

Install requirements from [eve](https://github.com/lf-edge/eve#install-dependencies)

Also, you need to install ```openssl``` package and ```uuidgen```

## Running

Recommended to run from superuser

To run harness use: ```make run```

To run tests use: ```make test```

To stop harness use: ```make stop```

## Help

You can see help by running ```make help```

## Object model
[Object model \#1](api/OM1.md)
