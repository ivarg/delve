package proctl

// #include "threads_darwin.h"
import "C"
import (
	"fmt"
	"unsafe"
)

type OSSpecificDetails struct {
	thread_act C.thread_act_t
}

func (t *ThreadContext) Halt() error {
	var kret C.kern_return_t
	kret = C.thread_suspend(t.os.thread_act)
	if kret != C.KERN_SUCCESS {
		return fmt.Errorf("could not suspend task %d", t.Id)
	}
	return nil
}

func (t *ThreadContext) singleStep() error {
	kret := C.single_step(t.os.thread_act)
	if kret != C.KERN_SUCCESS {
		return fmt.Errorf("could not single step")
	}
	trapWait(t.Process, 0)
	kret = C.clear_trap_flag(t.os.thread_act)
	if kret != C.KERN_SUCCESS {
		return fmt.Errorf("could not clear CPU trap flag")
	}
	return nil
}

func (t *ThreadContext) resume() error {
	// TODO(dp) set flag for ptrace stops
	if PtraceCont(t.Process.Pid, 0) == nil {
		return nil
	}
	for {
		if C.thread_resume(t.os.thread_act) != C.KERN_SUCCESS {
			break
		}
	}
	return nil
}

func (t *ThreadContext) blocked() bool {
	// TODO(dp) cache the func pc to remove this lookup
	pc, _ := t.CurrentPC()
	fn := t.Process.GoSymTable.PCToFunc(pc)
	if fn != nil && ((fn.Name == "runtime.mach_semaphore_wait") || (fn.Name == "runtime.usleep")) {
		return true
	}
	return false
}

func writeMemory(thread *ThreadContext, addr uintptr, data []byte) (int, error) {
	var (
		vm_data = unsafe.Pointer(&data[0])
		vm_addr = C.mach_vm_address_t(addr)
		length  = C.mach_msg_type_number_t(len(data))
	)

	if ret := C.write_memory(thread.Process.os.task, vm_addr, vm_data, length); ret < 0 {
		return 0, fmt.Errorf("could not write memory")
	}
	return len(data), nil
}

func readMemory(thread *ThreadContext, addr uintptr, data []byte) (int, error) {
	var (
		vm_data = unsafe.Pointer(&data[0])
		vm_addr = C.mach_vm_address_t(addr)
		length  = C.mach_msg_type_number_t(len(data))
	)

	ret := C.read_memory(thread.Process.os.task, vm_addr, vm_data, length)
	if ret < 0 {
		return 0, fmt.Errorf("could not read memory")
	}
	return len(data), nil
}
