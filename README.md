# clone-run

This repository contains a script written in `Go` lang, which will clone results from a test run in one project to another, as per the mapping of case Ids between the two projects.

The script uses the following environment variables:

- `QASE_API_TOKEN` - to be defined in your [repository secrets](https://docs.github.com/en/actions/security-for-github-actions/security-guides/using-secrets-in-github-actions#creating-secrets-for-a-repository). You can generate a token from [here](https://app.qase.io/user/api/token).
- `QASE_SOURCE_PROJECT` - the project where auto-test results are posted.
- `QASE_TARGET_PROJECT` - your project where manual + auto tests are managed.
- `QASE_SOURCE_RUN` - id of the auto-test run, in the source project.
- `QASE_TARGET_RUN` - id of the manual test run, in the target project.
- `QASE_CF_ID` - This shall be the custom field's Id in the target project.

[This article](https://help.qase.io/en/articles/9787250-how-do-i-find-my-project-code) can help in locating the project codes, or run ids.

<br>

## How to use?
- First, we'll need to map the test cases in the target project to corresponding test cases in the source project. Create a custom field at [this page](https://developers.qase.io/reference/create-custom-field). Instructions [here](https://imgur.com/delete/V3zS84Zl6zXJ9Hm).
- Once, the mapping is complete, you can trigger the script from the Actions tab of this repository.
  - There are three workflows available.
  - For all three workflows, variables `QASE_CF_ID`, `QASE_SOURCE_PROJECT` and `QASE_TARGET_PROJECT` can be defined directly in the [fallback.txt](./fallback.txt) file. You can always override the vlaues defined here, while triggering the workflow manually.
 
### Three workflows
1. [latest-run.yml](./.github/workflows/latest-run.yml): This automatically clones the results from the *latest* run in the source project to the *latest* run in the target project.
2. [specify-run.yml](./.github/workflows/specify-run.yml): You can specify the `source` and `target` run ids manually, while starting the workflow.
3. [trigger-from-qase.yml](./.github/workflows/trigger-from-qase.yml): Same as [1] – it assumes the latest run ids for both the projects, but this workflow can be triggered directly from Qase.

> When starting the third workflow from Qase, make sure to NOT trigger the workflow from either the `source` or `target` project.

<br>

## Gist of how the script works
1. Fetches test cases from source and target projects in Qase.
2. Maps test cases between the source and target projects using a custom field (from target project).
3. Fetches test results from a source test run.
4. Prepares these results for bulk creation in a target test run, including mapping test case IDs, attaching metadata, and converting status codes.
5. Writes a mapping file to a CSV for future reference.
6. Sends the prepared results to the target test run via the Qase API in bulk.

**Mapping Test Cases**: The script maps source test cases to their corresponding target test cases using a custom field value. It prepares an in-memory CSV with the mapping details, which includes: Source Case ID and Target Case ID.
