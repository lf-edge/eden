# Eden

Eden is where [EVE](https://github.com/lf-edge/eve) and [Adam](https://github.com/lf-edge/adam) get tried and tested.

Eden consists of a test harness and a series of integration tests implemented in Golang. Tests are structured as normal Golang tests by using ```_test.go``` nomenclature and be available for test runs using standard go test framework.

## Install Dependencies

Install requirements from [eve](https://github.com/lf-edge/eve#install-dependencies)

Also, you need to install ```uuidgen```

## Running

Recommended to run from superuser

To run harness use: ```make run```

To run tests use: ```make test```

To stop harness use: ```make stop```

## Help

You can see help by running ```make help```

## Utilites
   
   Utilites compile ```make bin```:
   * [ecerts](cmd/ecerts) -- SSL certificate generator;
   * [einfo](cmd/einfo) -- scans Info file accordingly by regular expression of requests to json fields;
   * [einfowatch](cmd/einfowatch) -- Info-files monitoring tool with regular expression quering to json fields;
   * [elog](cmd/elog) -- scans Log file accordingly by regular expression of requests to json fields;
   * [elogwatch](cmd/elogwatch) -- Log-files monitoring tool with regular expression quering to json fields; 
   * [eserver](cmd/eserver) -- micro HTTP-server for providing of baseOS and Apps images.
