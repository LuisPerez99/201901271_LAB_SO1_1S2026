KDIR := /usr/lib/modules/$(shell uname -r)/build
PWD  := $(shell pwd)
BUILD_DIR := $(PWD)/build

all:
	mkdir -p $(BUILD_DIR)/src
	cp src/main.c $(BUILD_DIR)/src/
	cp Kbuild $(BUILD_DIR)/
	make -C $(KDIR) M=$(BUILD_DIR) modules

clean:
	rm -rf $(BUILD_DIR)
