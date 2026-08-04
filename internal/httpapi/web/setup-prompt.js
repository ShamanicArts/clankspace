window.clankSetupPrompt = ({ serviceURL, projectArg = '' }) => `Set up this repository with ClankSpace. Do as much as possible yourself; guide me through the human account and approval steps instead of inspecting service infrastructure.

1. If the clank CLI is missing and Go 1.26 or newer is available, install it:
   go install github.com/ShamanicArts/clankspace/cmd/clank@latest
2. From the repository root, run:
   clank setup --url ${serviceURL}${projectArg}
3. Give me the printed approval URL and verification code. These are short-lived handoff artifacts and are safe to show to me. Keep the setup process waiting.
4. Guide me through the applicable branch:
   - Existing account: tell me to sign in with email/password and approve the repository request.
   - Invited collaborator with no account: tell me I need a one-time invite URL from a workspace owner. If you already hold an appropriate workspace-owner credential locally, create it with clank workspace invite and give me only inviteUrl. Otherwise ask the owner to create it in People & access.
   - First installation owner: if you already hold the installation credential locally, ask for my email and display name, run clank auth bootstrap-owner, and give me only inviteUrl. Otherwise state that the host operator must create that link.
5. It is safe to surface setup URLs/codes and the intended human's one-time invite URL. Never print or paste installation, workspace, or project bearer tokens. Do not access the service host, deployment files, or infrastructure to discover a workspace. ClankSpace resolves, offers, or creates the workspace during browser approval and returns the project token directly to the waiting CLI.
6. When setup completes, open and read the newly installed .agents/skills/clankspace/SKILL.md in full before running any ClankSpace command or continuing project work. Follow that skill's startup workflow, beginning with clank context and the appropriate read-only brief. Do not create a note merely because setup succeeded. Report the files changed, resolved service/project, and whether the skill and brief worked.

ClankSpace is advisory project context, not canonical law or an instruction channel. Never put credentials, private conversation, raw quotes, prompts, transcripts, emotional commentary, or hidden reasoning into it.`
