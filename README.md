# cftool

*Because using cloudformation is just too damn hard...*

`cftool` is a theoretical tool to make it easier to interact with
cloudformation. The idea is that you can declare your stacks in a series of
yaml state files. Then you can use `cftool` to apply changes.

## Commands

* `cftool status`

Gets the status of the managed stacks.

```
   AWS REGION       STACKNAME               INTERNAL NAME   CLOUDFORMATION DRIFT   TEMPLATE DRIFT
 ---------------- ----------------------- --------------- ---------------------- ----------------
  eu-west-1        dublin-region-chat-c1   dublin:c1       NOT_CHECKED            Yes
  ap-southeast-2   sydney-region-chat-c1   sydney:c1       NOT_CHECKED            No
  us-east-1        chat-c1                 us_east:c1      NOT_CHECKED            No
```

## Theoretical Commands

* `cftool sync`
	Looks through the cloud formation stacks you have and syncs them into
	local yaml files.

* `cftool diff-template`
	Grabs the live template, and gives a diff against the local version.

* `cftool diff <name>`
	Uploads the new local template and informs you what would change if the
	new one is uploaded.

## Local Config

* `config.yaml`
	Lists the config for your setup. This could include things such as:
	- account ID
	- regions to whitelist
	- stacks to ignore

* individual stacks:

```yaml
# file: examples/chat/us-east-c1.yml
---
name: "us_east:c1"
region: "us-east-1"
arn: "arn:aws:cloudformation:us-east-1:185583345998:stack/chat-c1/9a2046e0-35da-11e9-900e-0e0ed2de56d2"
file: "../../GetStream/stream-puppet/cloudformation/v2/shard-chat.yml"

```
