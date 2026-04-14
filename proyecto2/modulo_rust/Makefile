export KBUILD_MODPOST_WARN := 1
KDIR := $(HOME)/kernel-rust
PWD := $(shell pwd)

obj-m := src/modulo_rust.o

all:
	$(MAKE) -C $(KDIR) M=$(PWD) LLVM=1 modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) LLVM=1 clean