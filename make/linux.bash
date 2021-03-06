# Run from magnus root directory.
cd $GOPATH/src/github.com/runningwild/magnus

if CGO_LDFLAGS=-L/home/runningwild/code/src/github.com/runningwild/glop/gos/linux/lib/ go build --tags client . ; then
		rm -rf bin
		mkdir -p bin/fail

		cp magnus bin/fail
		cp ../glop/gos/linux/lib/libglop.so bin/fail/libglop.so
		echo "LD_LIBRARY_PATH=$LD_LIBRARY_PATH:. ./magnus" > bin/fail/runme
		chmod 776 bin/fail/runme
		cp -r data/* bin/

		cd bin/fail
		./runme
fi
