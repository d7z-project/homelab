import {
  Component,
  OnInit,
  Inject,
  inject,
  signal,
  ChangeDetectorRef,
  computed,
  effect,
  Optional,
  untracked,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormsModule,
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  Validators,
  FormArray,
  AbstractControl,
} from '@angular/forms';
import {
  MatDialog,
  MatDialogModule,
  MatDialogRef,
  MAT_DIALOG_DATA,
} from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatSelectModule } from '@angular/material/select';
import { MatIconModule } from '@angular/material/icon';
import { MatStepperModule } from '@angular/material/stepper';
import { MatDividerModule } from '@angular/material/divider';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatMenuModule } from '@angular/material/menu';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { DragDropModule, CdkDragDrop } from '@angular/cdk/drag-drop';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import {
  ActionsService,
  ModelsWorkflow,
  ModelsStep,
  ModelsStepManifest,
  ModelsVarDefinition,
} from '../../generated';
import { firstValueFrom } from 'rxjs';
import { ProcessorSelectorDialogComponent } from './processor-selector-dialog.component';
import { VariableConfigDialogComponent } from './variable-config-dialog.component';

import { DiscoverySelectComponent } from '../../shared/discovery-select.component';
import { DiscoverySuggestInputComponent } from '../../shared/discovery-suggest-input.component';

import { MonacoEditorModule } from 'ngx-monaco-editor-v2';
import * as yaml from 'js-yaml';

@Component({
  selector: 'app-create-workflow-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatSelectModule,
    MatIconModule,
    MatStepperModule,
    MatDividerModule,
    MatAutocompleteModule,
    MatToolbarModule,
    MatTooltipModule,
    MatSnackBarModule,
    MatCheckboxModule,
    MatSlideToggleModule,
    MatMenuModule,
    MatProgressSpinnerModule,
    DragDropModule,
    DiscoverySelectComponent,
    DiscoverySuggestInputComponent,
    MonacoEditorModule,
  ],
  templateUrl: './create-workflow-dialog.component.html',
  styles: [
    `
      :host {
        display: block;
        height: 100vh;
      }
      ::ng-deep .monaco-editor-container {
        height: 100% !important;
        width: 100% !important;
      }
      ::ng-deep .horizontal-stepper-full {
        height: 100%;
      }
      ::ng-deep .horizontal-stepper-full .mat-horizontal-stepper-wrapper {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-height: 0;
      }
      ::ng-deep .horizontal-stepper-full .mat-horizontal-content-container {
        flex: 1;
        padding: 0 !important;
        display: flex;
        flex-direction: column;
        min-height: 0;
      }
      ::ng-deep .horizontal-stepper-full .mat-horizontal-stepper-content {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-height: 0;
        outline: none !important;
      }
      ::ng-deep .mat-step-header {
        padding: 16px 24px !important;
        height: 72px !important;
      }
      ::ng-deep .mat-step-label {
        font-weight: 900 !important;
        text-transform: uppercase;
        letter-spacing: 1px;
        font-size: 12px !important;
      }
      @media (max-width: 600px) {
        ::ng-deep .mat-step-header {
          padding: 8px 12px !important;
        }
        ::ng-deep .mat-step-label {
          display: none; /* Hide labels on mobile to save space */
        }
      }
    `,
  ],
})
export class CreateWorkflowDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private orchService = inject(ActionsService);
  private cdr = inject(ChangeDetectorRef);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);

  editMode = signal<'visual' | 'yaml'>('visual');
  yamlCode = signal('');
  isEditorLoading = signal(true);
  monacoOptions = {
    theme: 'vs-dark',
    language: 'yaml',
    fontSize: 14,
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    renderLineHighlight: 'all',
    quickSuggestions: {
      other: true,
      comments: true,
      strings: true,
    },
    suggestOnTriggerCharacters: true,
    parameterHints: { enabled: true },
    formatOnType: true,
    tabSize: 2,
  };

  manifests = signal<ModelsStepManifest[]>([]);
  manifestMap = computed(() => {
    const map = new Map<string, ModelsStepManifest>();
    this.manifests().forEach((m) => {
      if (m.id) map.set(m.id, m);
    });
    return map;
  });

  schema = signal<any>(null);
  private completionProvider: any = null;

  infoForm: FormGroup = this.fb.group({
    name: ['', Validators.required],
    description: [''],
    enabled: [true],
    timeout: [7200, [Validators.required, Validators.min(0)]],
    serviceAccountId: ['', Validators.required],
    cronEnabled: [false],
    cronExpr: [''],
    webhookEnabled: [false],
  });

  vars: FormArray = this.fb.array([]);
  steps: FormArray = this.fb.array([]);

  // Create a signal from form changes to drive stepsData reactivity
  private stepsValue = toSignal(this.steps.valueChanges, { initialValue: [] });

  activeStepIndex = signal(0);
  totalSteps = computed(() => 4); // Fixed steps in visual mode

  // Pre-calculate step manifests to avoid function calls in template (NG0100 fix)
  stepsData = computed(() => {
    // Explicitly track both stepsValue signal and the raw array length
    this.stepsValue();
    const s = this.steps.controls as any[];
    return s.map((control) => {
      const type = control.get('type')?.value;
      const manifest = this.manifestMap().get(type);
      return {
        manifest,
        icon: manifest?.id?.startsWith('core/') ? 'memory' : 'api',
        params: manifest?.params || [],
      };
    });
  });

  constructor(
    public dialogRef: MatDialogRef<CreateWorkflowDialogComponent>,
    @Optional() @Inject(MAT_DIALOG_DATA) public data: { workflow: ModelsWorkflow | null } | null,
  ) {
    effect(() => {
      const s = this.schema();
      if (s) {
        untracked(() => this.applySchemaToMonaco(s));
      }
    });
  }

  async ngOnInit() {
    try {
      const [manifests, schema] = await Promise.all([
        firstValueFrom(this.orchService.actionsManifestsGet()),
        firstValueFrom(this.orchService.actionsWorkflowsSchemaGet()),
      ]);
      this.manifests.set(manifests || []);
      this.schema.set(schema);
      if (this.data?.workflow) {
        this.applyWorkflowToForms(this.data.workflow);
      }
    } catch (e) {
      console.error('Failed to load data', e);
    }
  }

  isValid() {
    if (this.editMode() === 'visual') {
      return this.infoForm.valid && this.vars.valid && this.steps.valid && this.steps.length > 0;
    }
    const code = this.yamlCode().trim();
    if (!code) return false;
    try {
      const parsed = yaml.load(code);
      return (
        !!parsed &&
        typeof parsed === 'object' &&
        (parsed as any).name &&
        Array.isArray((parsed as any).steps) &&
        (parsed as any).steps.length > 0
      );
    } catch (e) {
      return false;
    }
  }

  isStepValid(index: number): boolean {
    switch (index) {
      case 0:
        return this.infoForm.valid;
      case 1:
        return this.vars.valid;
      case 2:
        return this.steps.valid && this.steps.length > 0;
      case 3:
        return this.isValid();
      default:
        return true;
    }
  }

  onEditorInit(editor: any) {
    this.applySchemaToMonaco(this.schema());
    setTimeout(() => this.isEditorLoading.set(false));
  }

  private applySchemaToMonaco(schema: any) {
    const monaco = (window as any).monaco;
    if (!monaco || !schema) return;

    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      schemas: [
        {
          uri: 'homelab://schemas/workflow',
          fileMatch: ['*'],
          schema: schema,
        },
      ],
    });

    if (this.completionProvider) {
      this.completionProvider.dispose();
    }

    this.completionProvider = monaco.languages.registerCompletionItemProvider('yaml', {
      triggerCharacters: [
        ':',
        '-',
        ' ',
        'a',
        'b',
        'c',
        'd',
        'e',
        'f',
        'g',
        'h',
        'i',
        'j',
        'k',
        'l',
        'm',
        'n',
        'o',
        'p',
        'q',
        'r',
        's',
        't',
        'u',
        'v',
        'w',
        'x',
        'y',
        'z',
      ],

      provideCompletionItems: (model: any, position: any) => {
        const word = model.getWordUntilPosition(position);
        const lineContent = model.getLineContent(position.lineNumber);
        const hasDashOnLine = lineContent.trim().startsWith('-');

        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        const textBefore = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        const suggestions: any[] = [];

        const isTopLevel = !lineContent.startsWith(' ') && !textBefore.includes('  ');
        if (isTopLevel || lineContent.trim() === '') {
          Object.entries(schema.properties || {}).forEach(([key, val]: [string, any]) => {
            suggestions.push({
              label: key,
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: key + ': ',
              documentation: val.description,
              range: range,
              sortText: '0' + key,
            });
          });
        }

        if (textBefore.includes('steps:')) {
          this.manifests().forEach((m) => {
            const shortId = m.id?.split('/').pop();
            const prefix = hasDashOnLine ? '' : '- ';
            let snippet = `${prefix}id: \${1:${shortId}}\n  type: ${m.id}\n  fail: false\n  params:\n`;
            if (m.params) {
              m.params.forEach((p, i) => {
                snippet += `    ${p.name}: \${${i + 2}:""} # ${p.description || ''}\n`;
              });
            }

            suggestions.push({
              label: `step: ${m.name}`,
              kind: monaco.languages.CompletionItemKind.Snippet,
              insertText: snippet,
              insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              documentation: m.description,
              range: range,
              sortText: '1' + m.id,
            });
          });
        }

        return { suggestions };
      },
    });
  }

  switchMode(newMode: 'visual' | 'yaml') {
    if (this.editMode() === newMode) return;
    if (newMode === 'yaml') {
      const cleaned = this.getCurrentWorkflow();
      this.yamlCode.set(yaml.dump(cleaned, { indent: 2, noArrayIndent: true }));
    } else {
      try {
        const parsed = yaml.load(this.yamlCode()) as ModelsWorkflow;
        if (parsed) this.applyWorkflowToForms(parsed);
      } catch (e: any) {
        this.snackBar.open('YAML 解析失败: ' + e.message, '确定', { duration: 3000 });
        return;
      }
    }
    this.editMode.set(newMode);
    this.cdr.markForCheck();
  }

  getCurrentWorkflow(): ModelsWorkflow {
    const workflowValue = { ...this.infoForm.value };
    const varsMap: { [key: string]: ModelsVarDefinition } = {};

    this.vars.controls.forEach((control) => {
      const v = control.value;
      if (v.key) {
        varsMap[v.key] = {
          description: v.description,
          default: v.default,
          required: v.required,
          regexFrontend: v.regexFrontend,
          regexBackend: v.regexBackend,
        };
      }
    });

    const steps = this.steps.controls.map((control) => {
      const s = { ...control.value };
      if (!s.id || !s.id.trim()) {
        s.id = this.generateRandomId();
        control.get('id')?.setValue(s.id);
      }
      return s;
    });

    const workflow: ModelsWorkflow = {
      ...workflowValue,
      vars: Object.keys(varsMap).length > 0 ? varsMap : undefined,
      steps: steps,
      id: this.data?.workflow?.id,
    };

    return this.cleanObject(workflow);
  }

  private cleanObject(obj: any): any {
    if (obj === null || obj === undefined || obj === '' || obj === false || obj === 0) {
      return undefined;
    }

    if (Array.isArray(obj)) {
      const result = obj.map((item) => this.cleanObject(item)).filter((v) => v !== undefined);
      return result.length > 0 ? result : undefined;
    }

    if (typeof obj === 'object') {
      const cleanedObj: any = {};
      let hasVisibleData = false;

      for (const [key, value] of Object.entries(obj)) {
        const cleanedValue = this.cleanObject(value);
        if (cleanedValue === undefined) continue;

        cleanedObj[key] = cleanedValue;
        hasVisibleData = true;
      }
      return hasVisibleData ? cleanedObj : undefined;
    }

    return obj;
  }

  private generateRandomId(prefix = 'step_'): string {
    return prefix + Math.random().toString(36).substring(2, 8);
  }

  applyWorkflowToForms(wf: ModelsWorkflow) {
    this.infoForm.patchValue({
      name: wf.name,
      description: wf.description,
      enabled: wf.enabled,
      timeout: wf.timeout || 7200,
      serviceAccountId: wf.serviceAccountId,
      cronEnabled: wf.cronEnabled,
      cronExpr: wf.cronExpr,
      webhookEnabled: wf.webhookEnabled,
    });

    this.vars.clear();
    if (wf.vars) {
      Object.entries(wf.vars).forEach(([key, v]) => {
        this.vars.push(
          this.fb.group({
            key: [key, [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
            description: [v.description || ''],
            default: [v.default || ''],
            required: [v.required || false],
            regexFrontend: [v.regexFrontend || ''],
            regexBackend: [v.regexBackend || ''],
          }),
        );
      });
    }

    this.steps.clear();
    if (wf.steps) {
      wf.steps.forEach((s, idx) => {
        this.steps.push(
          this.fb.group({
            id: [
              s.id || this.generateRandomId(),
              [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)],
            ],
            type: [s.type, Validators.required],
            name: [s.name || ''],
            if: [s.if || ''],
            fail: [s.fail || false],
            params: this.fb.group({}),
          }),
        );
        this.onProcessorChange(idx, s.params);
      });
    }
  }

  getVarGroup(index: number) {
    return this.vars.at(index) as FormGroup;
  }
  addVar() {
    this.vars.push(
      this.fb.group({
        key: ['', [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
        description: [''],
        default: [''],
        required: [false],
        regexFrontend: [''],
        regexBackend: [''],
      }),
    );
  }
  removeVar(index: number) {
    this.vars.removeAt(index);
  }
  hasRegex(index: number) {
    const g = this.getVarGroup(index);
    return !!(g.get('regexFrontend')?.value || g.get('regexBackend')?.value);
  }

  openVarExtra(index: number) {
    const group = this.getVarGroup(index);
    this.dialog
      .open(VariableConfigDialogComponent, {
        width: '500px',
        data: {
          regexFrontend: group.get('regexFrontend')?.value,
          regexBackend: group.get('regexBackend')?.value,
        },
      })
      .afterClosed()
      .subscribe((res) => {
        if (res) {
          group.patchValue(res);
          this.cdr.markForCheck();
        }
      });
  }

  getStepGroup(index: number) {
    return this.steps.at(index) as FormGroup;
  }
  addStep() {
    const idx = this.steps.length;
    this.steps.push(
      this.fb.group({
        id: [this.generateRandomId(), [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
        type: ['', Validators.required],
        name: [''],
        if: [''],
        fail: [false],
        params: this.fb.group({}),
      }),
    );
    setTimeout(() => this.openProcessorSelector(idx));
  }
  removeStep(index: number) {
    this.steps.removeAt(index);
  }
  onStepDropped(e: CdkDragDrop<any[]>) {
    const temp = this.steps.at(e.previousIndex);
    this.steps.removeAt(e.previousIndex);
    this.steps.insert(e.currentIndex, temp);
  }

  openProcessorSelector(index: number) {
    const group = this.getStepGroup(index);
    this.dialog
      .open(ProcessorSelectorDialogComponent, {
        width: '600px',
        data: { manifests: this.manifests(), selectedId: group.get('type')?.value },
      })
      .afterClosed()
      .subscribe((id) => {
        if (id) {
          group.get('type')?.setValue(id);
          this.onProcessorChange(index);
          this.cdr.markForCheck();
        }
      });
  }

  getStepManifest(index: number) {
    const type = this.getStepGroup(index)?.get('type')?.value;
    return type ? this.manifestMap().get(type) : undefined;
  }

  onProcessorChange(index: number, initialParams?: any) {
    const group = this.getStepGroup(index);
    if (!group) return;
    const manifest = this.manifestMap().get(group.get('type')?.value);
    const paramsGroup = group.get('params') as FormGroup;
    Object.keys(paramsGroup.controls).forEach((k) => paramsGroup.removeControl(k));
    if (manifest?.params) {
      manifest.params.forEach((p) => {
        if (p.name) {
          const validators = p.optional ? [] : [Validators.required];

          // Add custom regex validator if defined
          if (p.regexFrontend) {
            validators.push((c: AbstractControl) => {
              const v = c.value;
              // Bypass regex check if it contains a variable placeholder
              if (!v || v.includes('${{')) return null;
              try {
                const re = new RegExp(p.regexFrontend!);
                return re.test(v) ? null : { regexMatch: true };
              } catch (e) {
                return null; // Invalid regex in manifest, don't block user
              }
            });
          }

          paramsGroup.addControl(
            p.name,
            this.fb.control(initialParams?.[p.name] || '', validators),
          );
        }
      });
    }
    // Force a value change notification to trigger computed signals
    this.steps.updateValueAndValidity();
  }

  getProcessorParams(type: string | undefined) {
    return type ? this.manifestMap().get(type)?.params || [] : [];
  }

  getOutputReferences(currentIndex: number) {
    const varRefs = this.vars.value
      .filter((v: any) => v.key)
      .map((v: any) => `\${{ vars.${v.key} }}`);

    const stepRefs = Array.from({ length: currentIndex }).flatMap((_, i) => {
      const g = this.getStepGroup(i);
      const id = g?.get('id')?.value;
      if (!id) return [];

      const refs = [`\${{ steps.${id}.status }}`];
      const manifest = this.manifestMap().get(g.get('type')?.value);

      manifest?.outputParams?.forEach((op) => {
        if (op.name) refs.push(`\${{ steps.${id}.outputs.${op.name} }}`);
      });
      return refs;
    });

    return [...varRefs, ...stepRefs];
  }

  async submit() {
    let workflow: ModelsWorkflow;
    try {
      if (this.editMode() === 'yaml') {
        workflow = yaml.load(this.yamlCode()) as ModelsWorkflow;
      } else {
        workflow = this.getCurrentWorkflow();
      }
      if (!workflow?.name || !workflow?.steps) throw new Error('工作流名称和步骤必填');

      // 1. Validate
      await firstValueFrom(this.orchService.actionsWorkflowsValidatePost(workflow));

      // 2. Save (Create or Update)
      if (this.data?.workflow?.id) {
        await firstValueFrom(
          this.orchService.actionsWorkflowsIdPut(this.data.workflow.id, workflow),
        );
        this.snackBar.open('工作流已更新', '确定', { duration: 3000 });
      } else {
        await firstValueFrom(this.orchService.actionsWorkflowsPost(workflow));
        this.snackBar.open('工作流已创建', '确定', { duration: 3000 });
      }

      this.dialogRef.close(true);
    } catch (e: any) {
      this.snackBar.open('操作失败: ' + (e.error?.message || e.message), '确定', {
        duration: 5000,
      });
    }
  }
}
