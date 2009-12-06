# prerequisite: GOROOT and GOARCH must be defined

# defines $(GC) (compiler), $(LD) (linker) and $(O) (architecture)
include $(GOROOT)/src/Make.$(GOARCH)

# name of the package (library) being built
TARG=gongo

# source files in package
GOFILES=\
	gongo_gtp.go \
	gongo_robot.go

# test files for this package
GOTESTFILES=\
	gongo_gtp_test.go \
	gongo_robot_test.go

all: main benchmark

# build "main" executable
main: package
	$(GC) -I_obj main.go
	$(LD) -L_obj -o $@ main.$O
	@echo "Done. Executable is: $@"

# build "benchmark" executable
benchmark: package
	$(GC) -I_obj benchmark.go
	$(LD) -L_obj -o $@ benchmark.$O
	@echo "Done. Executable is: $@"

clean:
	rm -rf *.[$(OS)o] *.a [$(OS)].out _obj _test _testmain.go main

package: _obj/$(TARG).a


# create a Go package file (.a)
_obj/$(TARG).a: _go_.$O
	@mkdir -p _obj/$(dir)
	rm -f _obj/$(TARG).a
	gopack grc $@ _go_.$O

# create Go package for the tests
_test/$(TARG).a: _gotest_.$O
	@mkdir -p _test/$(dir)
	rm -f _test/$(TARG).a
	gopack grc $@ _gotest_.$O

# compile
_go_.$O: $(GOFILES)
	$(GC) -o $@ $(GOFILES)

# compile tests
_gotest_.$O: $(GOFILES) $(GOTESTFILES)
	$(GC) -o $@ $(GOFILES) $(GOTESTFILES)


# targets needed by gotest

importpath:
	@echo $(TARG)

testpackage: _test/$(TARG).a

testpackage-clean:
	rm -f _test/$(TARG).a _gotest_.$O
