package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/nicolasgere/knit/lib/utils"
)

func NewRunner(ctx context.Context, concurency int) Runner {
	return Runner{
		semaphore: make(chan struct{}, concurency),
		ctx:       ctx,
	}
}

type Runner struct {
	semaphore chan struct{}
	ctx       context.Context
}

func (r *Runner) ExecCommand(cmd *exec.Cmd, tf *TaskFuture, task *Task) {
	r.semaphore <- struct{}{}
	defer func() { <-r.semaphore }()
	utils.LogTaskStart(task.Id, task.Cmd)
	pipeout, err := cmd.StdoutPipe()
	if err != nil {
		tf.Done <- TaskResult{Err: err, Status: 1}
	}
	ReaderToChan(&pipeout, tf.Stdout)
	pipeerr, err := cmd.StderrPipe()
	if err != nil {
		tf.Done <- TaskResult{Err: err, Status: 1}
	}
	ReaderToChan(&pipeerr, tf.Stderr)
	if err := cmd.Start(); err != nil {
		tf.Done <- TaskResult{Err: err, Status: 1}
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println("ERROR")
		if exiterr, ok := err.(*exec.ExitError); ok {
			tf.Done <- TaskResult{Err: err, Status: exiterr.ExitCode()}
		} else {
			tf.Done <- TaskResult{Err: err, Status: exiterr.ExitCode()}
		}
	}
	tf.Done <- TaskResult{Err: err, Status: 0}
}

func (r *Runner) RunTask(task Task) (tf *TaskFuture) {
	cmd := exec.CommandContext(r.ctx, "sh", "-c", task.Cmd)
	cmd.Dir = task.Root
	tf = &TaskFuture{
		Id:     task.Id,
		Stdout: make(chan []byte),
		Stderr: make(chan []byte),
		Done:   make(chan TaskResult, 1),
	}

	go r.ExecCommand(cmd, tf, &task)
	return
}

func (r *Runner) RunTasks(tasks []Task) (tf []*TaskFuture) {
	tf = make([]*TaskFuture, 0)
	for _, task := range tasks {
		tf = append(tf, r.RunTask(task))
	}
	return
}

func ReaderToChan(r *io.ReadCloser, out chan []byte) {
	go func() {
		rc := *r
		defer close(out)
		defer rc.Close()
		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			t := scanner.Bytes()
			out <- t
		}
		if err := scanner.Err(); err != nil {
			// Handle error (if needed)
			fmt.Println("Error reading:", err)
		}
	}()
}
