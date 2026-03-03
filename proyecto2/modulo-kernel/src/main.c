#include <linux/init.h>
#include <linux/kernel.h>
#include <linux/module.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched/signal.h>
#include <linux/mm.h>
#include <linux/sysinfo.h>
#include <linux/jiffies.h>
#include <linux/math64.h>
#include <linux/string.h>
#ifdef CONFIG_CGROUPS
#include <linux/cgroup.h>
#endif

#define PROC_NAME "continfo_pr2_so1_201901271"

static struct proc_dir_entry *proc_entry;

static void get_cmd_or_container(struct task_struct *task, char *out, size_t out_sz)
{
    char comm[TASK_COMM_LEN];
    get_task_comm(comm, task);
    strscopy(out, comm, out_sz);
}

static int continfo_show(struct seq_file *m, void *v)
{
    struct sysinfo info;
    unsigned long long total_ram, free_ram, used_ram; // ram in bytes
    unsigned long long uptime_j = get_jiffies_64();
    struct task_struct *task;

    si_meminfo(&info);

    total_ram = info.totalram * info.mem_unit / (1024 * 1024); // Convert to megabytes
    free_ram = info.freeram * info.mem_unit / (1024 * 1024); // Convert to megabytes
    used_ram = total_ram - free_ram; // Calculate used RAM

    seq_printf(m, "Memoria total: %llu MB, Memoria libre: %llu MB, Memoria usada: %llu MB\n", total_ram, free_ram, used_ram);
    
    seq_puts(m, "PID\tNAME\tCMD_OR_CONTAINER\tVSZ_KB\tRSS_KB\tMEM_PCT\tCPU_PCT\n");

    for_each_process(task) {
        struct mm_struct *mm;
        unsigned long vsz_kb, rss_kb;
        unsigned long long mem_pct_x100, cpu_pct;
        unsigned long long proc_time;
        char cmd_or_container[128];

        get_cmd_or_container(task, cmd_or_container, sizeof(cmd_or_container));

        mm = get_task_mm(task);
        if (mm) {
            vsz_kb = mm->total_vm << (PAGE_SHIFT - 10); // Convert to KB
            rss_kb = get_mm_rss(mm) << (PAGE_SHIFT - 10); // Convert to KB
            mmput(mm);
        }

        if (total_ram > 0) {
            mem_pct_x100 = div64_ul((unsigned long long)rss_kb * 10000, total_ram * 1024); // mem_pct * 100
        } else {
            mem_pct_x100 = 0;
        }
        seq_printf(m, "%d\t%s\t%s\t%lu\t%lu\t%llu.%02llu%%\t%llu\n",
                   task->pid,
                   task->comm,
                   cmd_or_container,
                   vsz_kb,
                   rss_kb,
                   mem_pct_x100 / 100, mem_pct_x100 % 100,
                   cpu_pct);
    }

    return 0;
}
