obj-m += src/continfo.o

BUILD_DIR := build

all:
	mkdir -p $(BUILD_DIR)
	$(MAKE) -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules
	cp -f *.ko *.o *.mod *.mod.c Module.symvers modules.order $(BUILD_DIR)/ 2>/dev/null || true
	cp -f src/*.ko $(BUILD_DIR)/ 2>/dev/null || true
	cp -f .*.cmd $(BUILD_DIR)/ 2>/dev/null || true
	mkdir -p $(BUILD_DIR)/src
	cp -f src/*.o src/*.mod.c $(BUILD_DIR)/src/ 2>/dev/null || true
	cp -f src/.*.cmd $(BUILD_DIR)/src/ 2>/dev/null || true
	rm -f *.ko *.o *.mod *.mod.c .*.cmd Module.symvers modules.order
	rm -f src/*.ko src/*.o src/*.mod src/*.mod.c src/.*.cmd
	rm -f .module-common.o

clean:
	$(MAKE) -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean
	rm -rf $(BUILD_DIR)
