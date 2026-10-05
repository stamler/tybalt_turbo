<script lang="ts">
  import { goto } from "$app/navigation";
  import type { loadClientEditor } from "$lib/clientEditorLoad";
  import { savedReturnTo } from "$lib/clientEditors";
  import { selectClientWorkflowProfile } from "$lib/clientWorkflowNavigation";
  import { pb } from "$lib/pocketbase";
  import type { WorkflowRecord } from "$lib/clientEditors";
  import ClientWorkspaceShell from "$lib/components/ClientWorkspaceShell.svelte";
  import ContactEditor from "./ContactEditor.svelte";
  import InvoicingEditor from "./InvoicingEditor.svelte";
  let { data }: { data: Awaited<ReturnType<typeof loadClientEditor>> } = $props();

  async function savedProfile(
    record: WorkflowRecord,
    mode?: "create" | "update" | "copy" | "reuse",
  ) {
    const workflowReturn = new URL(data.returnTo, "https://workspace.invalid").searchParams.get(
      "workflow_return",
    );
    const destination =
      mode === "create" || mode === "copy" || mode === "reuse"
        ? selectClientWorkflowProfile(
            workflowReturn,
            pb.authStore.record?.id || "",
            data.client.id,
            record.id,
          )
        : null;
    if (!destination && mode === "reuse") {
      const workspace = new URL(data.returnTo, "https://workspace.invalid");
      workspace.searchParams.delete("saved_contact");
      workspace.searchParams.delete("saved_invoicing");
      workspace.searchParams.delete("edit");
      workspace.searchParams.set("show_invoicing", record.id);
      workspace.hash = "invoicing";
      await goto(workspace.pathname + workspace.search + workspace.hash);
      return;
    }
    await goto(destination || savedReturnTo(data.returnTo, "invoicing", record.id));
  }
</script>

<ClientWorkspaceShell clientId={data.client.id} clientName={data.client.name}>
  <div class="mx-auto w-full max-w-3xl p-3">
    {#key `${data.client.id}/${data.kind}/${data.record.id || "new"}`}
      {#if data.kind === "contact"}
        <ContactEditor
          clientId={data.client.id}
          returnTo={data.returnTo}
          record={data.record}
          onSaved={(record) => goto(savedReturnTo(data.returnTo, "contact", record.id))}
          onCancel={() => goto(data.returnTo)}
          onDeleted={() => goto(data.returnTo)}
        />
      {:else}
        <InvoicingEditor
          client={data.client}
          contacts={data.client.contacts}
          record={data.record}
          projectContact={data.projectContact}
          onSaved={savedProfile}
          onCancel={() => goto(data.returnTo)}
        />
      {/if}
    {/key}
  </div>
</ClientWorkspaceShell>
