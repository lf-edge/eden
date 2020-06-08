#!/bin/sh

DIR=`mktemp -d --suffix=.eden_template_test`

#echo Test dir: $DIR
cp template_test_scenario.tmpl $DIR
cp /bin/echo $DIR

../../eden test $DIR -s template_test_scenario.tmpl > $DIR/out1

ROOT=`../../eden config get --key eden.root`
DIST=`../../eden config get --key eden.images.dist`
echo eden.root = $ROOT > $DIR/out2
echo eden.images.dist = $DIST '->' $ROOT/$DIST >> $DIR/out2
echo eden.images.dist = $DIST '->' $ROOT/$DIST >> $DIR/out2

if diff $DIR/out1 $DIR/out2
then
	echo --- PASS: TestTemplateSceanrio
	echo PASS
else
	echo --- FAIL: TestTemplateSceanrio
	echo FAIL
fi
rm -rf $DIR
