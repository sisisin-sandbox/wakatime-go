import * as pulumi from '@pulumi/pulumi';
import * as gcp from '@pulumi/gcp';
import * as std from '@pulumi/std';

export function jobPostBody(sa: gcp.serviceaccount.Account) {
  const config = pulumi.all([sa.email]).apply(([email]) => {
    const body = postBody(email);
    return JSON.stringify(body);
  });

  return std.base64encodeOutput({ input: config }).result;
}

export function postBody(email: string, commands: string[] = []) {
  return {
    taskGroups: [
      {
        taskSpec: {
          runnables: [
            { container: { imageUri: 'sisisin/wakatime-go:20240303-122753', entrypoint: '/app/main', commands } },
          ],
          environment: {
            secretVariables: { WAKATIME_KEY: 'projects/260114795237/secrets/wakatime-wakatime-key/versions/1' },
          },
          computeResource: { cpuMilli: 500, memoryMib: 200 },
          maxRetryCount: 1,
          maxRunDuration: '3600s',
        },
        taskCount: 1,
        parallelism: 1,
      },
    ],
    allocationPolicy: { instances: [{ policy: { machineType: 'e2-micro' } }], serviceAccount: { email } },
    labels: {},
    logsPolicy: { destination: 'CLOUD_LOGGING' },
  };
}
