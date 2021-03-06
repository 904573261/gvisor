package(licenses = ["notice"])  # Apache 2.0

load("//tools/go_generics:defs.bzl", "go_template_instance")
load("//tools/go_stateify:defs.bzl", "go_library", "go_stateify", "go_test")

go_stateify(
    name = "mm_state",
    srcs = [
        "aio_context.go",
        "aio_context_state.go",
        "file_refcount_set.go",
        "io_list.go",
        "mm.go",
        "pma_set.go",
        "save_restore.go",
        "special_mappable.go",
        "vma_set.go",
    ],
    out = "mm_state.go",
    package = "mm",
)

go_template_instance(
    name = "file_refcount_set",
    out = "file_refcount_set.go",
    imports = {
        "platform": "gvisor.googlesource.com/gvisor/pkg/sentry/platform",
    },
    package = "mm",
    prefix = "fileRefcount",
    template = "//pkg/segment:generic_set",
    types = {
        "Key": "uint64",
        "Range": "platform.FileRange",
        "Value": "int32",
        "Functions": "fileRefcountSetFunctions",
    },
)

go_template_instance(
    name = "vma_set",
    out = "vma_set.go",
    consts = {
        "minDegree": "8",
    },
    imports = {
        "usermem": "gvisor.googlesource.com/gvisor/pkg/sentry/usermem",
    },
    package = "mm",
    prefix = "vma",
    template = "//pkg/segment:generic_set",
    types = {
        "Key": "usermem.Addr",
        "Range": "usermem.AddrRange",
        "Value": "vma",
        "Functions": "vmaSetFunctions",
    },
)

go_template_instance(
    name = "pma_set",
    out = "pma_set.go",
    consts = {
        "minDegree": "8",
    },
    imports = {
        "usermem": "gvisor.googlesource.com/gvisor/pkg/sentry/usermem",
    },
    package = "mm",
    prefix = "pma",
    template = "//pkg/segment:generic_set",
    types = {
        "Key": "usermem.Addr",
        "Range": "usermem.AddrRange",
        "Value": "pma",
        "Functions": "pmaSetFunctions",
    },
)

go_template_instance(
    name = "io_list",
    out = "io_list.go",
    package = "mm",
    prefix = "io",
    template = "//pkg/ilist:generic_list",
    types = {
        "Linker": "*ioResult",
    },
)

go_library(
    name = "mm",
    srcs = [
        "address_space.go",
        "aio_context.go",
        "aio_context_state.go",
        "debug.go",
        "file_refcount_set.go",
        "io.go",
        "io_list.go",
        "lifecycle.go",
        "metadata.go",
        "mm.go",
        "mm_state.go",
        "pma.go",
        "pma_set.go",
        "proc_pid_maps.go",
        "save_restore.go",
        "shm.go",
        "special_mappable.go",
        "syscalls.go",
        "vma.go",
        "vma_set.go",
    ],
    importpath = "gvisor.googlesource.com/gvisor/pkg/sentry/mm",
    visibility = ["//pkg/sentry:internal"],
    deps = [
        "//pkg/abi/linux",
        "//pkg/atomicbitops",
        "//pkg/log",
        "//pkg/refs",
        "//pkg/sentry/arch",
        "//pkg/sentry/context",
        "//pkg/sentry/fs",
        "//pkg/sentry/fs/proc/seqfile",
        "//pkg/sentry/kernel/shm",
        "//pkg/sentry/limits",
        "//pkg/sentry/memmap",
        "//pkg/sentry/platform",
        "//pkg/sentry/platform/safecopy",
        "//pkg/sentry/safemem",
        "//pkg/sentry/usage",
        "//pkg/sentry/usermem",
        "//pkg/state",
        "//pkg/sync",
        "//pkg/syserror",
        "//pkg/tcpip/buffer",
    ],
)

go_test(
    name = "mm_test",
    size = "small",
    srcs = ["mm_test.go"],
    embed = [":mm"],
    deps = [
        "//pkg/sentry/arch",
        "//pkg/sentry/context",
        "//pkg/sentry/context/contexttest",
        "//pkg/sentry/limits",
        "//pkg/sentry/memmap",
        "//pkg/sentry/platform",
        "//pkg/sentry/usermem",
        "//pkg/syserror",
    ],
)
