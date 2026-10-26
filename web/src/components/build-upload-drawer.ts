import {html, LitElement} from 'lit';
import {customElement, property, state} from 'lit/decorators.js';
import {createRef, Ref, ref} from 'lit/directives/ref.js';
import {map} from 'lit/directives/map.js';

@customElement('build-upload-drawer')
export class BuildUploadDrawer extends LitElement {
    override createRenderRoot() {
        return this;
    }

    extensionsByPlatform = {
        "Windows": [".appx", ".appxbundle", ".appxupload", ".msix", ".msixbundle", ".msixupload", ".exe", ".zip", ".msi"],
        "macOS": [".zip", ".platform.zip", ".dmg", ".pkg"],
        "Linux": [".AppImage", ".deb", ".rpm",  ".tgz", ".gz", ".snap", ".flatpak"],
        "Android": [".apk", ".aab"],
        "iOS": [".ipa"],
    };

    archsByPlatform = {
        "Windows": ["x86", "x64"],
        "macOS": ["x64", "ARM64"],
        "Linux": ["x86", "x64", "ARM", "ARM64"],
        "Android": [""],
        "iOS": [""],
    };

    @property({type: String})
    slug = "";

    @property({type: String})
    platform = "Windows";

    @property({type: Boolean})
    showDrawer = false;

    @state()
    submitFailed = false;

    @state()
    uploading = false;

    errorAlertRef: Ref<HTMLDivElement> = createRef();

    @state()
    errorData: ErrorResponse = {message: '', helpTexts: []};

    handleSubmit(event: SubmitEvent) {
        event.preventDefault();
        if (this.uploading) return;
        this.submitFailed = false;

        void this.uploadBuild(event.currentTarget as HTMLFormElement);
    }

    async uploadBuild(form: HTMLFormElement) {
        this.uploading = true;
        const formData = new FormData(form);
        const files = Array.from(form.querySelectorAll<HTMLInputElement>('input[type="file"]'))
            .flatMap(input => Array.from(input.files ?? []).map(file => ({field: input.name, file})));
        const uploadID = crypto.randomUUID().replaceAll('-', '');
        const uploadBase = `/internal/apps/${this.slug}/platforms/${this.platform}/builds/chunks/${uploadID}`;
        const uploadDetails: {
            number: string;
            version: string;
            files: Record<string, {filename: string; size: number; chunks: number}>;
        } = {
            number: String(formData.get('Number') ?? ''),
            version: String(formData.get('Version') ?? ''),
            files: {},
        };

        try {
            for (const {field, file} of files) {
                const chunkCount = Math.ceil(file.size / (1 << 20));
                uploadDetails.files[field] = {filename: file.name, size: file.size, chunks: chunkCount};

                for (let index = 0; index < chunkCount; index++) {
                    const chunk = file.slice(index * (1 << 20), (index + 1) * (1 << 20));
                    const response = await fetch(`${uploadBase}/${encodeURIComponent(field)}/${index}`, {
                        method: 'PUT',
                        body: chunk,
                        credentials: 'include',
                    });
                    if (!response.ok) {
                        throw new Error(await this.readError(response));
                    }
                }
            }

            const response = await fetch(`${uploadBase}/complete`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(uploadDetails),
                credentials: 'include',
            });
            if (!response.ok) {
                throw new Error(await this.readError(response));
            }

            window.location.href = `/dashboard/${this.slug}/platforms/${this.platform}/builds/${encodeURIComponent(uploadDetails.number)}`;
        } catch (error) {
            this.submitFailed = true;
            this.errorData = {message: error instanceof Error ? error.message : 'Upload failed', helpTexts: []};
            await this.updateComplete;
            this.errorAlertRef.value?.scrollIntoView();
        } finally {
            this.uploading = false;
        }
    }

    async readError(response: Response) {
        const body = await response.text();
        try {
            const data = JSON.parse(body);
            return data.message || data.error || `Upload failed (${response.status})`;
        } catch {
            return body || `Upload failed (${response.status})`;
        }
    }

    handleClose() {
        this.showDrawer = false;
        this.submitFailed = false;
    }

    override render() {
        return html`
            <button type="button"
                    class="inline-flex items-center rounded-md bg-violet-900 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-violet-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-900"
                    @click="${() => this.showDrawer = true}">
                <i class="fa-solid fa-upload"></i>&nbsp;&nbsp;Upload
            </button>

            <headless-drawer ?open="${this.showDrawer}"
                             @closed="${this.handleClose}"
                             headerTitle="Upload Build">
                <form slot="body" id="build-upload-form" class="px-4 sm:px-6 pb-4"
                      @reset="${this.handleClose}"
                      @submit="${this.handleSubmit}">
                    <div class="space-y-6 pb-5 pt-6">
                        <div>
                            <label for="project-name" class="block text-sm/6 font-medium text-gray-900">Number</label>
                            <div class="mt-2">
                                <input type="text"
                                       name="Number"
                                       id="Number"
                                       class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-violet-800 sm:text-sm/6"
                                       maxlength="255"
                                       autocomplete="off"
                                       required>
                            </div>
                        </div>
                        <div>
                            <label for="project-name" class="block text-sm/6 font-medium text-gray-900">Version</label>
                            <div class="mt-2">
                                <input type="text"
                                       name="Version"
                                       id="Version"
                                       class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-violet-800 sm:text-sm/6"
                                       maxlength="255"
                                       autocomplete="off"
                                       required>
                            </div>
                        </div>

                        <div>
                            ${
                                    // @ts-ignore 
                                    map(this.archsByPlatform[this.platform], (arch) => html`
                                        <div class="mt-2">
                                            <label for="File${arch ? '_' + arch : ''}"
                                                   class="block text-sm/6 font-medium text-gray-900">Artifact
                                                ${arch}</label>
                                            <input type="file"
                                                   name="File${arch ? '_' + arch : ''}"
                                                   id="File${arch ? '_' + arch : ''}"
                                                   accept="${
                                                           // @ts-ignore 
                                                           this.extensionsByPlatform[this.platform].join(',')
                                                   }"
                                                   class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-violet-800 sm:text-sm/6"
                                                   maxlength="255"
                                                   autocomplete="off">
                                            <p class="text-xs/5 text-gray-600">Upload ${
                                                    // @ts-ignore 
                                                    this.extensionsByPlatform[this.platform].join(', ')
                                            } file(s)</p>
                                        </div>`
                                    )}
                        </div>
                    </div>

                    ${this.submitFailed ? html`
                        <error-alert ${ref(this.errorAlertRef)}>
                            <span slot="message">${this.errorData.message}</span>
                            <ul role="list" class="list-disc space-y-1 pl-5 mt-2" slot="description">
                                ${map(this.errorData.helpTexts, (text) => html`
                                    <li>${text}</li>`
                                )}
                            </ul>
                        </error-alert>` : ``}
                </form>

                <span slot="footer">
          <button type="reset"
                  form="build-upload-form"
                  class="rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50">
            Cancel
          </button>
          <button type="submit"
                  form="build-upload-form"
                                    ?disabled="${this.uploading}"
                  class="ml-4 inline-flex justify-center rounded-md bg-violet-800 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-violet-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-800">
                        ${this.uploading ? 'Uploading...' : 'Upload'}
          </button>
        </span>
            </headless-drawer>
        `;
    }
}

declare global {
    interface HTMLElementTagNameMap {
        'build-upload-drawer': BuildUploadDrawer;
    }
}
