# cftool

*Because using cloudformation is just too damn hard...*

`cftool` is a theoretical tool to make it easier to interact with
cloudformation. The idea is that you can declare your stacks in a series of
yaml state files. Then you can use `cftool` to apply changes.

## Theoretical Commands

* `cftool sync`
	Looks through the cloud formation stacks you have and syncs them into
	local yaml files.


## Local Config

* `config.yaml`
	Lists the config for your setup. This could include things such as:
	- account ID
	- regions to whitelist
	- stacks to ignore

* individual stacks:

```yaml
# file: us-east-c1.yml
name: us-east-c1
file: ../../cloudformation/shard-chat.yml
region: us-east-1
params:
  foo: bar
```
