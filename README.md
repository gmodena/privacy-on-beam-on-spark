# privacy-on-beam-on-spark
Testing Differential Privacy pipelines on Beam and PortableRunner.

This repo is a work in progress and contains a collection of notes and sample code to run `privacy-on-beam` modules
atop Beam PortableRunner.

#### Current status

`WorksOnMyComputer`.

A modified `pbeam` example compiles and is submitted to Spark (local, standalone, cluster).

1.The current run time depends on docker being available on the machine that will compile and submit the Beam pipeline (via a Job Server). As it stands now, this would be a no go at WMF.. I *think* that with some tinkering we could drop the docker dep (which seems required only for job submission), but this needs further investigation. 
2.The beam Go SDK I/O is predicated around google cloud or local filesystem access. To the best of my knowledge there currently is no off the shelf capability to write to HDFS from Go. The doc is not 100% clear on this. I'll need to dig a bit more into the source.


#### TODO
 - [ ] Test the job with actual I/O (e.g. test reading from file and writing to local/gs filesystem from Spark), and remove hardcoded examples.
 - [ ] Investigate running the job server without docker.
 - [ ] Investigate how the toolchain works. How is Go SDK executed on JVM? 
 - [ ] Investigate integration with HDFS, or other Hadoop facilities.
 - [ ] Investigate Flink.

# privacy-on-beam on spark
`beam.go` is a copy of `pbeam` doc's example modified to run atop (generic) runners instead of `direct` mode.
This is achieved by adding a dependency on github.com/apache/beam/sdks/go/pkg/beam/x/beamx. This packages
provides a wrapper around a number of runners.

`beamx` introduces a number of cli flags, including `--runner`. These can be parsed with:
```
// Parse beamx flags (e.r. --runner)
flag.Parse()
```

After a `Runtime` has been setup (see below), install deps and compile `beam.go` with

```bash
go get github.com/google/differential-privacy/privacy-on-beam/pbeam

```

The pipeline can then be initialised via `beamx.Run`
```go
// Wrapper around a number of runners, configurable via the
// --runner flag
if err := beamx.Run(context.Background(), p); err != nil {
	log.Fatalf("Failed to execute job: %v", err)
}
```

The pipeline can then be submitted to Spark with
```
./beam --runner PortableRunner --endpoint localhost:8099
go build beam.go
```

Where `--runner` indicates our runtime target, and `--endpoint` is the URL of a Job Server service that will ship
jobs to it.

The pipeline can run on a `direct` runner with:
```
./beam --runner direct
```

# Runtime

The following prerequisites are needed:
* Go SDK
* A local clone of `git@github.com:apache/beam.git`

## Go SDK Quickstart 
Follow the doc at https://beam.apache.org/get-started/quickstart-go/

The following will install Go SDK and the `wordcount.go` example.
```bash
go get -u github.com/apache/beam/sdks/go/...

export GOLANG_PROTOBUF_REGISTRATION_CONFLICT=ignore
go install github.com/apache/beam/sdks/go/examples/wordcount@latest
```

Verify that everything works as expected:
```bash
wordcount --input text --output counts
```

## Install and launch PortableRunner

A Docker image for running the PortableRunner Job Server is available on Dockerhub. Since I'm interested in exploring
the feasibility of a docker-less setup, I'll set things up from source.

```
git clone git@github.com:apache/beam.git
cd beam
./gradlew :runners:spark:2:job-server:runShadow
```
Will run spark, in-memory, in a docker container.

### Standalone local cluster
The in-memory container does not allow for much introspection. I'd like to access Spark UI, history, and logs.

The following will setup a local Spark running in standalone mode:

```bash
curl https://downloads.apache.org/spark/spark-2.4.8/spark-2.4.8-bin-hadoop2.7.tgz -o spark-2.4.8-bin-hadoop2.7.tgz
tar xvfj spark-2.4.8-bin-hadoop2.7.tgz
cd spark-2.4.8-bin-hadoop2.7
./sbin/start-all.sh
```
The latter command will deploy a master/slave setup. It requires `sshd` for inter-service comunication (e.g. on macOS, you'll have to turn on Remote Login).

On an occasion I had the Beam job fail due to insufficient resources on Spark. In that case, kiil (eventually) running spark processe and spin up master and slave manually with:
```bash
./sbin/start-master.sh
./sbin/start-slave.sh -m 2048M -c 2
```
Where `-m` indicates how much memory to allocate to the worker process, and `-c` the number of cores.

Finally, we can instruct Beam to submit jobs to the local cluster by specifying the spark master url:
```
./gradlew :runners:spark:2:job-server:runShadow add  -PsparkMasterUrl=spark://localhost:7077 
```
