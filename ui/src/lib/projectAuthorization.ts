export const projectAuthorizationNotApprovedCode = "project_authorization_not_approved";
export const projectAuthorizationEntryMessage = "This job is missing an approved PA document.";
const genericProjectManagerPhrase = "the project's manager";

type FieldErrorWithCode = {
  message?: string;
  code?: string;
  [key: string]: unknown;
};

type FieldErrorsWithCodes = Record<string, FieldErrorWithCode>;

type BlockingProjectAuthorizationJob = {
  id: string;
  manager_name: string;
};

function projectAuthorizationMessageWithManager(message: string, managerName: string): string {
  if (message.includes(genericProjectManagerPhrase)) {
    return message.replace(genericProjectManagerPhrase, managerName);
  }
  if (message.toLowerCase().includes(managerName.toLowerCase())) {
    return message;
  }
  const separator = /[.!?]\s*$/.test(message) ? " " : ". ";
  return `${message}${separator}Speak with ${managerName}.`;
}

export function withProjectAuthorizationManagerName<T extends FieldErrorsWithCodes>(
  fieldErrors: T,
  managerName: string | null | undefined,
): T {
  const trimmedManagerName = managerName?.trim() ?? "";
  const jobError = fieldErrors.job;
  if (
    trimmedManagerName === "" ||
    jobError?.code !== projectAuthorizationNotApprovedCode ||
    typeof jobError.message !== "string"
  ) {
    return fieldErrors;
  }

  return {
    ...fieldErrors,
    job: {
      ...jobError,
      message: projectAuthorizationMessageWithManager(jobError.message, trimmedManagerName),
    },
  } as T;
}

export function projectAuthorizationBlockingJobs(
  error: unknown,
): Map<string, BlockingProjectAuthorizationJob> {
  const payload = (error as { response?: any; data?: any })?.response ?? (error as any)?.data;
  const blockingJobs =
    payload?.code === projectAuthorizationNotApprovedCode ? payload.blocking_jobs : [];
  if (!Array.isArray(blockingJobs)) return new Map();

  return new Map(
    blockingJobs.flatMap((job: unknown) => {
      if (typeof job !== "object" || job === null) return [];
      const { id, manager_name } = job as { id?: unknown; manager_name?: unknown };
      if (typeof id !== "string") return [];
      return [
        [
          id,
          {
            id,
            manager_name: typeof manager_name === "string" ? manager_name.trim() : "",
          },
        ],
      ];
    }),
  );
}
