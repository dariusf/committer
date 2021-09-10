#!/usr/bin/env python

import subprocess
import pathlib
import os
import time
import shutil
import re

def mkdirp(path):
  pathlib.Path(path).mkdir(parents=True, exist_ok=True)

def start_master(f, n):
  urls = ','.join(f'localhost:300{i+1}' for i in range(n))
  followers = f'-followers={urls}'
  return subprocess.Popen(['./committer', '-role=coordinator', '-nodeaddr=localhost:3000', followers, '-committype=two-phase', '-timeout=1000', '-dbpath=/tmp/badger/coordinator', '-whitelist=127.0.0.1'],
    stdout=f,
    stderr=subprocess.STDOUT,
    cwd=os.getcwd())

def start_replica(f, n):
  i = n + 1
  return subprocess.Popen(['./committer', '-role=follower', f'-nodeaddr=localhost:300{i}', '-committype=two-phase', '-timeout=1000', f'-dbpath=/tmp/badger/follower{i}', '-whitelist=127.0.0.1'],
    stdout=f,
    stderr=subprocess.STDOUT,
    cwd=os.getcwd())

def start_client(reqs, f):
  return subprocess.Popen(['examples/client/client'],
    stdout=f,
    stderr=subprocess.STDOUT,
    # TODO
    env={'REQUESTS': str(reqs)},
    cwd=os.getcwd())

def clean(replica_count):
  shutil.rmtree('/tmp/badger', ignore_errors=True)
  shutil.rmtree('out', ignore_errors=True)
  mkdirp('out')
  mkdirp('/tmp/badger/coordinator')
  for i in range(replica_count):
    mkdirp(f'/tmp/badger/follwer{i}')

def build():
  subprocess.check_call(['go', 'build'],
    stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)
  subprocess.check_call(['go', 'build'], cwd='examples/client')

def run_experiment(reqs, replica_count):
  replica_files = []
  replica_processes = []
  with open('out/master.log', 'w') as master_f, open('out/client.log', 'w') as client_f:
    master_process = start_master(master_f, replica_count)
    for i in range(replica_count):
      fr = open(f'out/replica{i}.log', 'w')
      replica_files.append(fr)
      replica_processes.append(start_replica(fr, i))

    print('waiting for processes to start')
    time.sleep(3)
    start_client(reqs, client_f).wait()
    print('client terminated')

    for f in replica_files:
      f.close()
    for p in replica_processes:
      p.terminate()
      p.wait()

    master_process.terminate()
    master_process.wait()

def collect_data(replica_count):
  with open('out/client.log', 'r') as f:
    for m in re.finditer(r'Total time taken: (\d+)', f.read()):
      client_time = int(m.group(1))
      print('client time', client_time)

  monitor_time = 0
  with open('out/master.log', 'r') as f:
    for m in re.finditer(r'Monitor time taken: (\d+)', f.read()):
      t = int(m.group(1))
      monitor_time += t
      print('master time', t)

  for i in range(replica_count):
    with open(f'out/replica{i}.log', 'r') as f:
      for m in re.finditer(r'Monitor time taken: (\d+)', f.read()):
        t = int(m.group(1))
        monitor_time += t
        print('replica', i, 'time', t)

  print('monitor time', monitor_time)
  return monitor_time, client_time

def run_it_all(reqs, runs, replica_count):
  monitor_time = 0
  client_time = 0
  for i in range(runs):
    print(f'---- run {i}')
    clean(replica_count)
    run_experiment(reqs, replica_count)
    m, c = collect_data(replica_count)
    monitor_time += m
    client_time += c

  monitor_time /= runs
  client_time /= runs
  print(f'------')
  overhead = monitor_time / client_time
  print(f'avg overhead for {replica_count} replicas: {overhead}')


if __name__ == "__main__":
  build()

  # testing baby steps
  # runs = 1
  # runs = 2
  # replica_counts = {2}
  # replica_counts = {3}
  # reqs = 5

  # paper configuration
  runs = 5
  replica_counts = {2, 4, 6}
  reqs = 100
  for c in replica_counts:
    print(f'------ {c} replicas')
    run_it_all(reqs, runs, c)
