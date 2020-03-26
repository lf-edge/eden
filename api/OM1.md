# Object model \#1

## Swagger model
* [YAML spec.](api/swagger/eden.yaml)
* [Converter YAML spec. to PNG] api/swagger/swagger_to_png.sh


## Structures:
* EdgeDevConfig (eve/api/go/config/devconfig.pb.go)
* AppInstanceConfig (eve/api/go/config/appconfig.pb.go)
* NetworkConfig (eve/api/go/config/netconfig.pb.go)
* DatastoreConfig (eve/api/go/config/storage.pb.go)

## Base functions
* register -- Must send to Adam data related to Eve instance onboarding (Certificate, Serial). Function must return unique Eve instance UUID.
* config -- Set configuration for Eve instance by unique UUID. Such configutation must be described on struct EdgeDevConfig with arrays of structs: Apps []\*AppInstanceConfig, Networks []\*NetworkConfig and Datastores []\*DatastoreConfig.
* log -- Get logs and info data for Eve instance by unique UUID.

## CRUD functions
* createAppInstance
* stateAppInstance
* changeAppInstance
* deleteAppInstance
