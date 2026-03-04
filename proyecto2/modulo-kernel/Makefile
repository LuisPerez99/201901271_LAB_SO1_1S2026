obj-m += continfo.o
continfo-objs := src/main.o

KDIR := /usr/lib/modules/$(shell uname -r)/build
PWD  := $(shell pwd)
BUILD_DIR := $(PWD)/build

all:
	mkdir -p $(BUILD_DIR)
	make -C $(KDIR) M=$(BUILD_DIR) src=$(PWD) modules

clean:
	rm -rf $(BUILD_DIR)
