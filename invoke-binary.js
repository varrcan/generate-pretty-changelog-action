const childProcess = require('child_process');
const os = require('os');
const process = require('process');
const core = require('@actions/core');

function chooseBinary() {
	const platform = os.platform();
	const arch = os.arch();

	// otherwise assume it's a released version
	if (platform === 'linux' && arch === 'x64') {
		return `bin/generate-changelog_linux_amd64`;
	}
	if (platform === 'darwin' && arch === 'x64') {
		return `bin/generate-changelog_darwin_amd64`;
	}

	core.setFailed(`Unsupported platform (${platform}) and architecture (${arch})`);
}

function main() {
	const binary = chooseBinary();
	const mainScript = `${__dirname}/${binary}`;
	const spawnSyncReturns = childProcess.spawnSync(mainScript, {stdio: 'inherit'});
	const status = spawnSyncReturns.status;
	if (status !== 0) {
		core.setFailed(spawnSyncReturns.error);
	}
}

if (require.main === module) {
	main();
}
