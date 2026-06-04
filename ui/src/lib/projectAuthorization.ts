const projectAuthorizationNotApprovedCode = "project_authorization_not_approved";
const genericProjectManagerPhrase = "the project's manager";

type FieldErrorWithCode = {
  message?: string;
  code?: string;
  [key: string]: unknown;
};

type FieldErrorsWithCodes = Record<string, FieldErrorWithCode>;

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
