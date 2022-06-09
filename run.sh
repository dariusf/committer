#!/bin/bash

set -x
set -e

p1='make run-example-follower1'
p2='make run-example-follower2'
c='make run-example-coordinator'
cl='make run-example-client'

clean() {
  ps aux | grep committer | grep -v grep | awk '{print $2}' | xargs kill
  tmux kill-session -t test_2pc || true
}

clean
make prepare

# https://stackoverflow.com/a/40009032

# -----------
# | p1 | p2 |
# |---------|
# | cl | c  |
# -----------

# tmux new-session -d -s test_2pc $SHELL
# tmux send-keys "$p2" ENTER
# tmux split-window -h
# tmux send-keys "$p1" ENTER
# tmux split-window -v
# tmux send-keys "$cl" #ENTER
# tmux select-pane -t 0
# tmux split-window -v
# tmux send-keys "$c" ENTER
# tmux select-pane -t 3

# tmux a

win=test_2pc
tmux new-window -n $win
tmux send-keys "$p2" ENTER
tmux split-window
tmux send-keys "$p1" ENTER
tmux split-window
tmux send-keys "$c" ENTER
tmux split-window
tmux select-layout tiled
tmux send-keys "$cl" ENTER
