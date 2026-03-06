import {
  Component,
  Inject,
  OnInit,
  inject,
  signal,
  ChangeDetectorRef,
  computed,
  effect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormsModule,
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  Validators,
  FormArray,
  ValidatorFn,
  AbstractControl,
  ValidationErrors,
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
import { MatMenuModule } from '@angular/material/menu';
import { DragDropModule, CdkDragDrop } from '@angular/cdk/drag-drop';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import {
  ActionsService,
  ModelsWorkflow,
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
    MatMenuModule,
    DragDropModule,
    DiscoverySelectComponent,
    DiscoverySuggestInputComponent,
    MonacoEditorModule,
  ],
  template: `
    <div class="flex flex-col h-full bg-surface-container-lowest overflow-hidden">
      <mat-toolbar
        class="!bg-surface !border-b !border-outline-variant/30 flex justify-between shrink-0 h-16"
      >
        <div class="flex items-center">
          <button mat-icon-button icon-button-center (click)="dialogRef.close()" matTooltip="返回">
            <mat-icon>close</mat-icon>
          </button>
          <span class="ml-2 text-lg font-medium tracking-tight">{{
            data.workflow ? '编辑工作流' : '创建新工作流'
          }}</span>
        </div>

        <div class="flex items-center gap-4">
          <div class="bg-surface-container-high p-1 rounded-full flex items-center shrink-0">
            @if (editMode() === 'visual') {
              <button mat-flat-button color="primary" class="!rounded-full !h-8 !min-w-[80px] sm:!min-w-[100px] !shadow-none">图形化</button>
              <button mat-button (click)="switchMode('yaml')" class="!rounded-full !h-8 !min-w-[80px] sm:!min-w-[100px]">YAML</button>
            } @else {
              <button mat-button (click)="switchMode('visual')" class="!rounded-full !h-8 !min-w-[80px] sm:!min-w-[100px]">图形化</button>
              <button mat-flat-button color="primary" class="!rounded-full !h-8 !min-w-[80px] sm:!min-w-[100px] !shadow-none">YAML</button>
            }
          </div>

          <button
            mat-button
            color="primary"
            (click)="submit()"
            [disabled]="!isValid()"
            class="!rounded-full font-bold"
          >
            保存
          </button>
        </div>
      </mat-toolbar>

      <div class="flex-1 flex flex-col min-h-0 relative" [class.p-4]="editMode() === 'visual'" [class.sm:p-8]="editMode() === 'visual'">
        @if (editMode() === 'visual') {
          <div class="max-w-4xl mx-auto w-full h-full overflow-y-auto animate-in fade-in duration-300">
            <mat-stepper orientation="vertical" #stepper class="!bg-transparent">
              <mat-step [stepControl]="infoForm">
                <ng-template matStepLabel>基本信息</ng-template>
                <form [formGroup]="infoForm" class="flex flex-col gap-4 mt-6 max-w-2xl">
                  <mat-form-field appearance="outline">
                    <mat-label>工作流名称</mat-label>
                    <input matInput formControlName="name" placeholder="例如：每日数据备份" />
                    <mat-error>名称必填</mat-error>
                  </mat-form-field>
                  <mat-form-field appearance="outline">
                    <mat-label>描述</mat-label>
                    <textarea matInput formControlName="description" rows="2" placeholder="对该工作流的简要说明"></textarea>
                  </mat-form-field>

                  <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <app-discovery-select
                      code="rbac/serviceaccounts"
                      label="执行身份 (ServiceAccount)"
                      placeholder="搜索账号 ID 或名称..."
                      formControlName="serviceAccountId"
                      hint="任务将以此身份权限运行"
                    ></app-discovery-select>

                    <mat-form-field appearance="outline">
                      <mat-label>超时时间 (秒)</mat-label>
                      <input matInput type="number" formControlName="timeout" placeholder="默认 7200" />
                      <mat-hint>0 为不超时</mat-hint>
                    </mat-form-field>
                  </div>

                  <div class="bg-surface-container-low p-6 rounded-2xl border border-outline-variant/30 space-y-4">
                    <div class="flex justify-between items-center">
                      <p class="text-xs font-bold uppercase tracking-wider text-outline">触发器配置</p>
                      <mat-checkbox formControlName="enabled">启用此工作流</mat-checkbox>
                    </div>
                    <div class="flex flex-col gap-4">
                      <mat-checkbox formControlName="cronEnabled">启用定时任务 (Cron)</mat-checkbox>
                      @if (infoForm.get('cronEnabled')?.value) {
                        <mat-form-field appearance="outline" class="ml-8">
                          <mat-label>Cron 表达式</mat-label>
                          <input matInput formControlName="cronExpr" placeholder="例如：0 0 * * *" />
                          <mat-hint>标准 Crontab 格式</mat-hint>
                        </mat-form-field>
                      }
                      <mat-checkbox formControlName="webhookEnabled">启用 Webhook 触发</mat-checkbox>
                    </div>
                  </div>
                  <div class="mt-2 flex gap-2">
                    <button mat-flat-button matStepperNext type="button" color="primary">下一步</button>
                  </div>
                </form>
              </mat-step>

              <mat-step>
                <ng-template matStepLabel>运行变量配置 (可选)</ng-template>
                <div class="flex flex-col gap-4 mt-6 max-w-3xl pb-8">
                  @for (v of vars.controls; track $index) {
                    @let vIndex = $index;
                    <div [formGroup]="getVarGroup(vIndex)" class="bg-surface-container-low p-4 sm:p-6 rounded-2xl border border-outline-variant/30 flex flex-col gap-4 relative">
                      <div class="flex justify-between items-start">
                        <span class="text-[10px] font-bold uppercase tracking-widest text-outline bg-outline/5 px-2 py-0.5 rounded">变量 #{{ vIndex + 1 }}</span>
                        <button mat-icon-button color="warn" (click)="removeVar(vIndex)" type="button"><mat-icon>delete_outline</mat-icon></button>
                      </div>
                      <div class="grid grid-cols-1 sm:grid-cols-12 gap-4">
                        <mat-form-field appearance="outline" class="sm:col-span-3"><mat-label>键名</mat-label><input matInput formControlName="key" /></mat-form-field>
                        <mat-form-field appearance="outline" class="sm:col-span-4"><mat-label>描述</mat-label><input matInput formControlName="description" /></mat-form-field>
                        <mat-form-field appearance="outline" class="sm:col-span-5"><mat-label>默认值</mat-label><input matInput formControlName="default" /></mat-form-field>
                      </div>
                      <div class="flex items-center gap-6">
                        <mat-checkbox formControlName="required">必填</mat-checkbox>
                        <button mat-button type="button" (click)="openVarExtra(vIndex)">
                          <mat-icon class="mr-2">{{ hasRegex(vIndex) ? 'verified_user' : 'tune' }}</mat-icon>
                          {{ hasRegex(vIndex) ? '已配置正则校验' : '配置正则校验' }}
                        </button>
                      </div>
                    </div>
                  }
                  <button mat-stroked-button color="primary" (click)="addVar()" type="button" class="border-dashed inline-flex items-center gap-2">
                    <mat-icon>add</mat-icon><span>添加变量</span>
                  </button>
                  <div class="mt-4 flex gap-2">
                    <button mat-button matStepperPrevious type="button">上一步</button>
                    <button mat-flat-button matStepperNext type="button" color="primary">下一步</button>
                  </div>
                </div>
              </mat-step>

              <mat-step>
                <ng-template matStepLabel>任务执行步骤 (Steps)</ng-template>
                <div class="flex flex-col gap-6 mt-6 pb-20" cdkDropList (cdkDropListDropped)="onStepDropped($event)">
                  @for (step of steps.controls; track $index) {
                    @let sIndex = $index;
                    <div [formGroup]="getStepGroup(sIndex)" cdkDrag class="bg-surface p-4 sm:p-6 rounded-2xl border border-outline-variant/30 shadow-sm flex flex-col gap-4 relative group">
                      <div class="flex justify-between items-center">
                        <div class="flex items-center gap-3">
                          <mat-icon cdkDragHandle class="cursor-grab">drag_indicator</mat-icon>
                          <span class="text-[10px] font-bold uppercase tracking-widest text-outline bg-outline/5 px-2 py-0.5 rounded">Step #{{ sIndex + 1 }}</span>
                        </div>
                        <button mat-icon-button color="warn" (click)="removeStep(sIndex)" type="button" class="opacity-0 group-hover:opacity-100 transition-opacity"><mat-icon>delete_outline</mat-icon></button>
                      </div>
                      <div class="grid grid-cols-1 sm:grid-cols-12 gap-4">
                        <div class="sm:col-span-12 flex flex-col sm:flex-row gap-4 items-start">
                          <button mat-stroked-button type="button" (click)="openProcessorSelector(sIndex)" class="!rounded-xl !h-14 flex items-center gap-3 min-w-[200px]">
                            <mat-icon color="primary">{{ getStepManifest(sIndex)?.id?.startsWith('core/') ? 'memory' : 'api' }}</mat-icon>
                            <div class="flex flex-col items-start leading-none">
                              <span class="text-xs text-outline mb-0.5 font-normal">执行处理器</span>
                              <span class="text-sm font-bold truncate max-w-[150px]">{{ getStepManifest(sIndex)?.name || '未选择' }}</span>
                            </div>
                            <mat-icon class="ml-auto">arrow_drop_down</mat-icon>
                          </button>
                          <mat-form-field appearance="outline" class="flex-1 w-full sm:w-auto"><mat-label>步骤 ID</mat-label><input matInput formControlName="id" /></mat-form-field>
                          <mat-form-field appearance="outline" class="flex-1 w-full sm:w-auto"><mat-label>显示名称 (可选)</mat-label><input matInput formControlName="name" /></mat-form-field>
                        </div>
                        <mat-form-field appearance="outline" class="sm:col-span-12">
                          <mat-label>执行条件 (If - 可选)</mat-label>
                          <input matInput formControlName="if" [matAutocomplete]="refAuto" />
                          <mat-autocomplete #refAuto="matAutocomplete">@for (ref of getOutputReferences(sIndex); track ref) { <mat-option [value]="ref">{{ ref }}</mat-option> }</mat-autocomplete>
                        </mat-form-field>
                        <div class="sm:col-span-12 bg-surface-container-highest/40 p-4 rounded-2xl border border-outline-variant/10">
                          <p class="text-[10px] font-bold uppercase tracking-widest text-outline/60 mb-4">参数配置</p>
                          <div formGroupName="params" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            @for (p of getProcessorParams(getStepGroup(sIndex).get('type')?.value); track p.name) {
                              <mat-form-field appearance="outline">
                                <mat-label>{{ p.description || p.name }}</mat-label>
                                @if (p.lookupCode) { <app-discovery-suggest-input [code]="p.lookupCode" [formControlName]="p.name!" /> }
                                @else { <input matInput [formControlName]="p.name!" [matAutocomplete]="refAuto" /> }
                              </mat-form-field>
                            } @empty { <p class="text-xs text-outline/40 italic">此处理器无需配置参数</p> }
                          </div>
                        </div>
                      </div>
                    </div>
                  }
                  <button mat-fab extended color="primary" (click)="addStep()" type="button" class="!rounded-2xl mx-auto"><mat-icon>add</mat-icon>添加任务步骤</button>
                  <div class="mt-4 flex gap-2"><button mat-button matStepperPrevious type="button">上一步</button><button mat-flat-button matStepperNext type="button" color="primary">下一步</button></div>
                </div>
              </mat-step>

              <mat-step>
                <ng-template matStepLabel>确认保存</ng-template>
                <div class="mt-8 p-8 bg-surface border border-outline-variant rounded-2xl text-center max-w-2xl mx-auto">
                  <mat-icon class="text-5xl h-auto w-auto text-primary opacity-20 mb-4">verified</mat-icon>
                  <h3 class="text-xl font-bold mb-2">准备就绪</h3>
                  <div class="flex justify-center gap-4"><button mat-button matStepperPrevious type="button">返回修改</button><button mat-flat-button color="primary" (click)="submit()" [disabled]="!isValid()">立即保存</button></div>
                </div>
              </mat-step>
            </mat-stepper>
          </div>
        } @else {
          <div class="flex-1 overflow-hidden animate-in fade-in duration-300 bg-[#1e1e1e]">
            <ngx-monaco-editor class="h-full w-full" style="height: 100%; width: 100%" [options]="monacoOptions" [(ngModel)]="yamlCode" (onInit)="onEditorInit($event)"></ngx-monaco-editor>
          </div>
        }
      </div>
    </div>
  `,
  styles: [`
    :host { display: block; height: 100vh; }
    ::ng-deep .monaco-editor-container { height: 100% !important; width: 100% !important; }
    ::ng-deep .mat-stepper-vertical { background: transparent !important; }
    ::ng-deep .mat-step-header { border-radius: 12px !important; margin-bottom: 8px !important; }
  `]
})
export class CreateWorkflowDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private orchService = inject(ActionsService);
  private cdr = inject(ChangeDetectorRef);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);

  editMode = signal<'visual' | 'yaml'>('visual');
  yamlCode = '';
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
    this.manifests().forEach(m => { if (m.id) map.set(m.id, m); });
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

  constructor(
    public dialogRef: MatDialogRef<CreateWorkflowDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { workflow: ModelsWorkflow | null },
  ) {
    effect(() => {
      const s = this.schema();
      if (s) { this.applySchemaToMonaco(s); }
    });
  }

  async ngOnInit() {
    try {
      const [manifests, schema] = await Promise.all([
        firstValueFrom(this.orchService.actionsManifestsGet()),
        firstValueFrom(this.orchService.actionsWorkflowsSchemaGet())
      ]);
      this.manifests.set(manifests || []);
      this.schema.set(schema);
      if (this.data.workflow) {
        this.applyWorkflowToForms(this.data.workflow);
      }
    } catch (e) { console.error('Failed to load data', e); }
  }

  isValid() {
    if (this.editMode() === 'visual') {
      return this.infoForm.valid && this.steps.length > 0;
    }
    if (!this.yamlCode.trim()) return false;
    try {
      const parsed = yaml.load(this.yamlCode);
      return !!parsed && typeof parsed === 'object' && (parsed as any).name && (parsed as any).steps;
    } catch (e) {
      return false;
    }
  }

  onEditorInit(editor: any) {
    this.applySchemaToMonaco(this.schema());
  }

  private applySchemaToMonaco(schema: any) {
    const monaco = (window as any).monaco;
    if (!monaco || !schema) return;

    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      schemas: [{
        uri: 'homelab://schemas/workflow',
        fileMatch: ['*'],
        schema: schema,
      }],
    });

    if (this.completionProvider) {
      this.completionProvider.dispose();
    }

    this.completionProvider = monaco.languages.registerCompletionItemProvider('yaml', {
      triggerCharacters: [':', '-', ' ', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'],
      
      provideCompletionItems: (model: any, position: any) => {
        const word = model.getWordUntilPosition(position);
        const lineContent = model.getLineContent(position.lineNumber);
        
        // 核心改进：检测当前行是否已有横杠
        const hasDashOnLine = lineContent.trim().startsWith('-');
        
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn
        };

        const textBefore = model.getValueInRange({
          startLineNumber: 1, startColumn: 1,
          endLineNumber: position.lineNumber, endColumn: position.column
        });
        
        const suggestions: any[] = [];
        
        // 顶级字段
        const isTopLevel = !lineContent.startsWith(' ') && !textBefore.includes('  ');
        if (isTopLevel || lineContent.trim() === '') {
          Object.entries(schema.properties || {}).forEach(([key, val]: [string, any]) => {
            suggestions.push({
              label: key,
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: key + ': ',
              documentation: val.description,
              range: range,
              sortText: '0' + key
            });
          });
        }

        // 步骤提示
        if (textBefore.includes('steps:')) {
           this.manifests().forEach(m => {
             const shortId = m.id?.split('/').pop();
             
             // 核心改进：如果已经有了横杠，snippet 不再带横杠
             const prefix = hasDashOnLine ? '' : '- ';
             let snippet = `${prefix}id: \${1:${shortId}}\n  type: ${m.id}\n  params:\n`;
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
               sortText: '1' + m.id
             });
           });
        }

        return { suggestions };
      }
    });
  }

  switchMode(newMode: 'visual' | 'yaml') {
    if (this.editMode() === newMode) return;
    if (newMode === 'yaml') {
      this.yamlCode = yaml.dump(this.getCurrentWorkflow(), { indent: 2, noArrayIndent: true });
    } else {
      try {
        const parsed = yaml.load(this.yamlCode) as ModelsWorkflow;
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
    const varsMap: { [key: string]: ModelsVarDefinition } = {};
    this.vars.value.forEach((v: any) => {
      if (v.key) {
        varsMap[v.key] = {
          description: v.description,
          default: v.default,
          required: v.required,
          regexFrontend: v.regexFrontend,
          regexBackend: v.regexBackend
        };
      }
    });
    return {
      ...this.infoForm.value,
      vars: varsMap,
      steps: this.steps.value,
      id: this.data.workflow?.id
    };
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
      webhookEnabled: wf.webhookEnabled
    });

    this.vars.clear();
    if (wf.vars) {
      Object.entries(wf.vars).forEach(([key, v]) => {
        this.vars.push(this.fb.group({
          key: [key, [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
          description: [v.description || ''],
          default: [v.default || ''],
          required: [v.required || false],
          regexFrontend: [v.regexFrontend || ''],
          regexBackend: [v.regexBackend || '']
        }));
      });
    }

    this.steps.clear();
    if (wf.steps) {
      wf.steps.forEach((s, idx) => {
        this.steps.push(this.fb.group({
          id: [s.id, [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
          type: [s.type, Validators.required],
          name: [s.name || ''],
          if: [s.if || ''],
          params: this.fb.group({})
        }));
        this.onProcessorChange(idx, s.params);
      });
    }
  }

  getVarGroup(index: number) { return this.vars.at(index) as FormGroup; }
  addVar() {
    this.vars.push(this.fb.group({
      key: ['', [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
      description: [''],
      default: [''],
      required: [false],
      regexFrontend: [''],
      regexBackend: ['']
    }));
  }
  removeVar(index: number) { this.vars.removeAt(index); }
  hasRegex(index: number) {
    const g = this.getVarGroup(index);
    return !!(g.get('regexFrontend')?.value || g.get('regexBackend')?.value);
  }

  openVarExtra(index: number) {
    const group = this.getVarGroup(index);
    this.dialog.open(VariableConfigDialogComponent, {
      width: '500px',
      data: {
        regexFrontend: group.get('regexFrontend')?.value,
        regexBackend: group.get('regexBackend')?.value
      }
    }).afterClosed().subscribe(res => {
      if (res) {
        group.patchValue(res);
        this.cdr.markForCheck();
      }
    });
  }

  getStepGroup(index: number) { return this.steps.at(index) as FormGroup; }
  addStep() {
    const idx = this.steps.length;
    this.steps.push(this.fb.group({
      id: ['', [Validators.required, Validators.pattern(/^[a-z0-9_]+$/)]],
      type: ['', Validators.required],
      name: [''],
      if: [''],
      params: this.fb.group({})
    }));
    this.openProcessorSelector(idx);
  }
  removeStep(index: number) { this.steps.removeAt(index); }
  onStepDropped(e: CdkDragDrop<any[]>) {
    const temp = this.steps.at(e.previousIndex);
    this.steps.removeAt(e.previousIndex);
    this.steps.insert(e.currentIndex, temp);
  }

  openProcessorSelector(index: number) {
    const group = this.getStepGroup(index);
    this.dialog.open(ProcessorSelectorDialogComponent, {
      width: '600px',
      data: { manifests: this.manifests(), selectedId: group.get('type')?.value }
    }).afterClosed().subscribe(id => {
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
    Object.keys(paramsGroup.controls).forEach(k => paramsGroup.removeControl(k));
    if (manifest?.params) {
      manifest.params.forEach(p => {
        if (p.name) {
          const validators = p.optional ? [] : [Validators.required];
          if (p.regexFrontend) {
            validators.push((c: AbstractControl) => {
              const v = c.value;
              if (!v || v.includes('${{')) return null;
              try { if (!new RegExp(p.regexFrontend!).test(v)) return { regexMatch: true }; } catch { return null; }
              return null;
            });
          }
          paramsGroup.addControl(p.name, this.fb.control(initialParams?.[p.name] || '', validators));
        }
      });
    }
  }

  getProcessorParams(type: string | undefined) {
    return type ? this.manifestMap().get(type)?.params || [] : [];
  }

  getOutputReferences(currentIndex: number) {
    const refs: string[] = [];
    this.vars.value.forEach((v: any) => { if (v.key) refs.push(`\${{ vars.${v.key} }}`); });
    for (let i = 0; i < currentIndex; i++) {
      const g = this.getStepGroup(i);
      const manifest = this.manifestMap().get(g?.get('type')?.value);
      if (g?.get('id')?.value && manifest?.outputParams) {
        manifest.outputParams.forEach(op => {
          if (op.name) refs.push(`\${{ steps.${g.get('id')?.value}.outputs.${op.name} }}`);
        });
      }
    }
    return refs;
  }

  async submit() {
    let workflow: ModelsWorkflow;
    try {
      if (this.editMode() === 'yaml') {
        workflow = yaml.load(this.yamlCode) as ModelsWorkflow;
      } else {
        workflow = this.getCurrentWorkflow();
      }
      if (!workflow?.name || !workflow?.steps) throw new Error('工作流名称和步骤必填');
      await firstValueFrom(this.orchService.actionsWorkflowsValidatePost(workflow));
      this.dialogRef.close(workflow);
    } catch (e: any) {
      this.snackBar.open('校验失败: ' + (e.error?.message || e.message), '确定', { duration: 5000 });
    }
  }
}
