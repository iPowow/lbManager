
# Updater components
This repository contains the following components.

## Updater library

The **Updater** library wraps all the logic to check for version update in CoreRoller for your own application.

	package main

	import (
		"os"
		"os/signal"
		"time"
		"syscall"

		"bitbucket.org/ipowow/updater"
	)

	func main() {
		// Instantiates the CoreRoller updater to check periodically for version update.
		if updater, err := updater.New(30*time.Second, syscall.SIGTERM); err == nil {
			go updater.Start()
		}

		// Your own application starts here...

		// Wait for termination signal
		signalsCh := make(chan os.Signal, 1)
		signal.Notify(signalsCh, os.Interrupt, syscall.SIGTERM)
		<-signalsCh
	}

We have packaged a sampleapp application in this repo that does that (see below).

## Artifact library

The **Artifact** library is a small helper library used by **Updater** library to download the artifact packages and install them on the host/container.

## Binitializer program

The **binitializer** (aka binary initializer) is a GO program that will contact CoreRoller to ask for the latest version of your application/package, download it and install it on your host/container. It uses the **Artifact** library to do so.

To run it requires the following environment variables and parameters:

**Environment Variables**

<table>
<tr>
    <td width="33%">CR_API_URL</td><td>API endpoint of CoreRoller (not Omaha endpoint).<br/>
    It uses the authenticated endpoint, and not Omaha so it bypasses the update policy restrictions.</td>
</tr><tr>
    <td width="33%">CR_INSTANCE_ID</td><td>Machine or Instance ID to identify your host.</td>
</tr><tr>
    <td width="33%"><i>&ltARTIFACT_PREFIX&gt</i>_CR_APP_ID</td><td>The application's UUID in CoreRoller for your artifact.</td>
</tr><tr>
    <td width="33%"><i>&ltARTIFACT_PREFIX&gt</i>_CR_GROUP_ID</td><td>The application's group UUID in CoreRoller for your artifact.</td>
</tr>
</table>

Note that the last two environment variables contain as prefix the name of the artifact in uppercase. Artifact refers to a deployable version of your package. So for example, if your application is called `Sample App`, your package might be called `sampleapp_v1.0.2`. In that case you would use `SAMPLEAPP` as ARTIFACT_PREFIX.

For example:

	export CR_API_URL=http://localhost:8000
	export SAMPLEAPP_CR_APP_ID=a829621b-6f8c-485e-822b-a586c7ebfeb2
	export SAMPLEAPP_CR_GROUP_ID=958e7b8f-eeb9-4568-8169-9a1ef5e88cfd

Then you can run it like that:

	go run main.go -artifacts sampleapp -dir /tmp

**Program parameters**

<table>
<tr>
    <td width="20%">-artifacts</td><td>One or more application/artifacts to install (comma separated).<br/>
Multiple applications can be specified since each environment variable starts with the name of the application.</td>
</tr><tr>
    <td width="20%">-dir</td><td>Directory where to install the binaries</td>
</tr>
</table>

## Sampleapp program

The **sampleapp** program as its name suggests is a sample application to demonstrate how easy it is to include version check and update in your own application. It uses the **Updater** library as explained at the top of this document.

To run it requires the following environment variables and parameters:

**Environment Variables**

<table>
<tr>
    <td width="33%">CR_OMAHA_URL</td><td>Omaha endpoint of CoreRoller (not API endpoint)</td>
</tr><tr>
    <td width="33%">CR_INSTANCE_ID</td><td>Machine or Instance ID to identify your host</td>
</tr><tr>
    <td width="33%"><i>&ltAPPLICATION&gt</i>_CR_APP_ID</td><td>Your application ID in CoreRoller</td>
</tr><tr>
    <td width="33%"><i>&ltAPPLICATION&gt</i>_CR_GROUP_ID</td><td>Your application's Group ID in CoreRoller</td>
</tr>
</table>


For example:

	export CR_OMAHA_URL=http://localhost:8000/omaha/
	export CR_INSTANCE_ID=instance-01
	export SAMPLEAPP_CR_APP_ID=a829621b-6f8c-485e-822b-a586c7ebfeb2
	export SAMPLEAPP_CR_GROUP_ID=958e7b8f-eeb9-4568-8169-9a1ef5e88cfd

Then you can run it like that:

	go run main.go

