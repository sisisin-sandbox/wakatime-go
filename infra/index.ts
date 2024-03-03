import * as pulumi from '@pulumi/pulumi';
import * as gcp from '@pulumi/gcp';
const gcpConfig = new pulumi.Config('gcp');
const project = gcpConfig.require('project');
const region = gcpConfig.require('region');
const config = new pulumi.Config();
const projectNumber = config.require('projectNumber');

const saCloudRunDownloader = new gcp.serviceaccount.Account('wakatime-cr-downloader', {
  accountId: 'wakatime-cr-downloader',
  displayName: 'Wakatime Downloader for Cloud Run',
});
const cloudRunDownloaderRoles = ['roles/storage.admin', 'roles/secretmanager.secretAccessor'];
applyIAMMember('wakatime-cr-downloader', saCloudRunDownloader, cloudRunDownloaderRoles);

const saSchedulerInvokeDownloader = new gcp.serviceaccount.Account('wakatime-scheduler-invoker', {
  accountId: 'wakatime-scheduler-invoker',
  displayName: 'Wakatime Scheduler Invoke Downloader',
});
const schedulerInvokeDownloaderRoles = ['roles/run.invoker'];
applyIAMMember('wakatime-scheduler-invoker', saSchedulerInvokeDownloader, schedulerInvokeDownloaderRoles);

const runName = 'wakatime-downloader';
new gcp.cloudscheduler.Job('wakatime-downloader-cr', {
  schedule: '0 1 * * *', // every day at 1:00
  timeZone: 'Asia/Tokyo',
  httpTarget: {
    httpMethod: 'POST',
    uri: `https://${region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${projectNumber}/jobs/${runName}:run`,
    headers: { 'Content-Type': 'application/json' },
    oauthToken: {
      serviceAccountEmail: saSchedulerInvokeDownloader.email,
    },
  },
});

function applyIAMMember(key: string, sa: gcp.serviceaccount.Account, roles: string[]) {
  roles.forEach((role) => {
    new gcp.projects.IAMMember(`${key}-${role}`, {
      project,
      role,
      member: pulumi.interpolate`serviceAccount:${sa.email}`,
    });
  });
}
