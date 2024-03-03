import * as pulumi from '@pulumi/pulumi';
import * as gcp from '@pulumi/gcp';
import * as std from '@pulumi/std';
import { jobPostBody } from './jobConfig';
const gcpConfig = new pulumi.Config('gcp');
const project = gcpConfig.require('project');
const region = gcpConfig.require('region');
const config = new pulumi.Config();
const projectNumber = config.require('projectNumber');

const saDownloader = new gcp.serviceaccount.Account('wakatime-downloader', {
  accountId: 'wakatime-downloader',
  displayName: 'Wakatime Downloader',
});

export const wakatimeDownloaderEmail = saDownloader.email;

const downloaderRoles = [
  'roles/batch.agentReporter',
  'roles/storage.admin',
  'roles/logging.logWriter',
  'roles/secretmanager.secretAccessor',
];
downloaderRoles.forEach((role) => {
  new gcp.projects.IAMMember(`wakatime-downloader-${role}`, {
    project,
    role,
    member: pulumi.interpolate`serviceAccount:${saDownloader.email}`,
  });
});

const saDownloadJobScheduler = new gcp.serviceaccount.Account('wakatime-download-scheduler', {
  accountId: 'wakatime-download-scheduler',
  displayName: 'Wakatime Download Job Scheduler',
});
const downloadJobSchedulerRoles = ['roles/batch.jobsEditor', 'roles/iam.serviceAccountUser'];
downloadJobSchedulerRoles.forEach((role) => {
  new gcp.projects.IAMMember(`wakatime-download-job-scheduler-${role}`, {
    project,
    role,
    member: pulumi.interpolate`serviceAccount:${saDownloadJobScheduler.email}`,
  });
});

new gcp.cloudscheduler.Job('wakatime-downloader', {
  schedule: '0 1 * * *', // every day at 1:00
  timeZone: 'Asia/Tokyo',
  httpTarget: {
    httpMethod: 'POST',
    uri: `https://batch.googleapis.com/v1/projects/${projectNumber}/locations/${region}/jobs`,
    body: jobPostBody(saDownloader),
    headers: { 'Content-Type': 'application/json' },
    oauthToken: {
      serviceAccountEmail: saDownloadJobScheduler.email,
      scope: 'https://www.googleapis.com/auth/cloud-platform',
    },
  },
});
