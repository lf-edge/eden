#!/bin/sh

DIR=`dirname $0`
UML=`echo $1| sed 's/\.yaml/\.puml/'`

# https://github.com/nlohmann/swagger_to_uml
echo "$DIR/swagger_to_uml/swagger_to_uml.py $1 > $UML"
$DIR/swagger_to_uml/swagger_to_uml.py $1 > $UML
# https://plantuml.com
echo "java -jar $DIR/plantuml.jar $UML"
java -jar $DIR/plantuml.jar $UML
