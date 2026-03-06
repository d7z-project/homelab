import {
  Component,
  Inject,
  OnInit,
  inject,
  signal,
  ChangeDetectorRef,
  computed,
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
  RbacService,
  ModelsWorkflow,
  ModelsStep,
  ModelsStepManifest,
  ModelsParamDefinition,
  ModelsServiceAccount,
  ModelsVarDefinition,
} from '../../generated';
import { firstValueFrom } from 'rxjs';
import { ProcessorSelectorDialogComponent } from './processor-selector-dialog.component';
import { VariableConfigDialogComponent } from './variable-config-dialog.component';

import { DiscoverySelectComponent } from '../../shared/discovery-select.component';
import { DiscoverySuggestInputComponent } from '../../shared/discovery-suggest-input.component';

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
  ],
  template: `
    <div class="flex flex-col h-full bg-surface-container-lowest overflow-hidden">
      <mat-toolbar
        class="!bg-surface !border-b !border-outline-variant/30 flex justify-between shrink-0 h-16 sm:h-16"
      >
        <div class="flex items-center">
          <button mat-icon-button icon-button-center (click)="dialogRef.close()" matTooltip="返回">
            <mat-icon>close</mat-icon>
          </button>
          <span class="ml-2 text-lg font-medium tracking-tight">{{
            data.workflow ? '编辑工作流' : '创建新工作流'
          }}</span>
        </div>
        <button
          mat-button
          color="primary"
          (click)="submit()"
          [disabled]="!infoForm.valid || steps.length === 0"
        >
          保存
        </button>
      </mat-toolbar>

      <div class="flex-1 overflow-y-auto p-4 sm:p-8">
        <div class="max-w-4xl mx-auto">
          <mat-stepper orientation="vertical" #stepper class="!bg-transparent">
            <!-- Step 1: Basic Info -->
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
                  <textarea
                    matInput
                    formControlName="description"
                    placeholder="对该工作流的简要说明"
                    rows="2"
                  ></textarea>
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
                    <input
                      matInput
                      type="number"
                      formControlName="timeout"
                      placeholder="默认 7200"
                    />
                    <mat-hint>0 为不超时，超时将自动终止</mat-hint>
                  </mat-form-field>
                </div>

                <div
                  class="bg-surface-container-low p-6 rounded-2xl border border-outline-variant/30 space-y-4"
                >
                  <div class="flex justify-between items-center">
                    <p class="text-xs font-bold uppercase tracking-wider text-outline">
                      触发器配置
                    </p>
                    <mat-checkbox formControlName="enabled">启用此工作流</mat-checkbox>
                  </div>

                  <div class="flex flex-col gap-4">
                    <div class="flex flex-col gap-2">
                      <mat-checkbox formControlName="cronEnabled">启用定时任务 (Cron)</mat-checkbox>
                      @if (infoForm.get('cronEnabled')?.value) {
                        <mat-form-field appearance="outline" class="ml-8">
                          <mat-label>Cron 表达式</mat-label>
                          <input
                            matInput
                            formControlName="cronExpr"
                            placeholder="例如：0 0 * * *"
                          />
                          <mat-hint>标准 Crontab 格式</mat-hint>
                        </mat-form-field>
                      }
                    </div>

                    <div class="flex flex-col gap-2">
                      <mat-checkbox formControlName="webhookEnabled"
                        >启用 Webhook 触发</mat-checkbox
                      >
                      @if (infoForm.get('webhookEnabled')?.value) {
                        <div class="ml-8 text-[10px] text-outline italic">
                          开启后，可通过唯一的 Token 异步触发此工作流。
                        </div>
                      }
                    </div>
                  </div>
                </div>

                <div class="mt-2 flex gap-2">
                  <button mat-flat-button matStepperNext type="button" color="primary">
                    下一步
                  </button>
                </div>
              </form>
            </mat-step>

            <!-- Step 2: Variables -->
            <mat-step>
              <ng-template matStepLabel>运行变量配置 (可选)</ng-template>
              <div class="flex flex-col gap-4 mt-6 max-w-3xl pb-8">
                <p class="text-sm text-outline mb-2">
                  声明工作流执行时需要接收的变量（如 Webhook 传参）。可以在后续步骤中引用它们。
                </p>

                @for (v of vars.controls; track $index) {
                  @let vIndex = $index;
                  <div
                    [formGroup]="getVarGroup(vIndex)"
                    class="bg-surface-container-low p-4 sm:p-6 rounded-2xl border border-outline-variant/30 flex flex-col gap-4 relative animate-in fade-in slide-in-from-top-2"
                  >
                    <div class="flex justify-between items-start">
                      <span
                        class="text-[10px] font-bold uppercase tracking-widest text-outline bg-outline/5 px-2 py-0.5 rounded"
                        >变量 #{{ vIndex + 1 }}</span
                      >
                      <button
                        mat-icon-button
                        icon-button-center
                        color="warn"
                        (click)="removeVar(vIndex)"
                        matTooltip="删除变量"
                        type="button"
                        class="!w-8 !h-8 -mt-2 -mr-2"
                      >
                        <mat-icon class="!text-lg">delete_outline</mat-icon>
                      </button>
                    </div>

                    <div class="grid grid-cols-1 sm:grid-cols-12 gap-4 items-start">
                      <mat-form-field appearance="outline" class="sm:col-span-3">
                        <mat-label>变量键名</mat-label>
                        <input matInput formControlName="key" placeholder="如：env" />
                        @if (getVarGroup(vIndex).get('key')?.errors?.['pattern']) {
                          <mat-error>小写字母、数字、下划线</mat-error>
                        }
                      </mat-form-field>

                      <mat-form-field appearance="outline" class="sm:col-span-4">
                        <mat-label>变量描述</mat-label>
                        <input
                          matInput
                          formControlName="description"
                          placeholder="此变量用途说明"
                        />
                      </mat-form-field>

                      <mat-form-field appearance="outline" class="sm:col-span-5">
                        <mat-label>默认值 (可选)</mat-label>
                        <input matInput formControlName="default" placeholder="未填时的回退值" />
                        @if (getVarGroup(vIndex).get('default')?.errors?.['regexMatch']) {
                          <mat-error>不符合前端正则</mat-error>
                        }
                      </mat-form-field>
                    </div>

                    <div
                      class="flex items-center justify-between pt-2 border-t border-outline-variant/10"
                    >
                      <div class="flex items-center gap-6">
                        <mat-checkbox formControlName="required" class="font-medium"
                          >设为必填参数</mat-checkbox
                        >
                        <div class="w-px h-4 bg-outline-variant/30 hidden sm:block"></div>
                        <button
                          mat-button
                          type="button"
                          (click)="openVarExtra(vIndex)"
                          [class.text-success]="hasRegex(vIndex)"
                          class="!rounded-full !px-4"
                        >
                          <mat-icon class="mr-2">{{
                            hasRegex(vIndex) ? 'verified_user' : 'tune'
                          }}</mat-icon>
                          {{ hasRegex(vIndex) ? '已配置正则校验' : '配置正则校验' }}
                        </button>
                      </div>
                    </div>
                  </div>
                }

                <div class="flex">
                  <button
                    mat-stroked-button
                    color="primary"
                    (click)="addVar()"
                    type="button"
                    class="border-dashed inline-flex items-center gap-2"
                  >
                    <mat-icon class="!m-0">add</mat-icon>
                    <span>添加变量</span>
                  </button>
                </div>

                <div class="mt-4 flex gap-2">
                  <button mat-button matStepperPrevious type="button">上一步</button>
                  <button mat-flat-button matStepperNext type="button" color="primary">
                    下一步
                  </button>
                </div>
              </div>
            </mat-step>

            <!-- Step 3: Configure Steps -->
            <mat-step>
              <ng-template matStepLabel>任务步骤配置</ng-template>
              <div
                class="flex flex-col gap-6 mt-6 pb-12"
                cdkDropList
                (cdkDropListDropped)="drop($event)"
              >
                @for (step of steps.controls; track step.value._uid) {
                  @let stepIndex = $index;
                  <div
                    cdkDrag
                    class="bg-surface border border-outline-variant rounded-2xl overflow-hidden shadow-sm transition-shadow hover:shadow-md relative"
                  >
                    <!-- Drag handle -->
                    <div
                      class="absolute left-0 top-0 bottom-0 w-1.5 bg-primary/10 hover:bg-primary transition-colors z-10 cursor-move"
                      cdkDragHandle
                      matTooltip="按住拖动排序"
                    ></div>

                    <!-- Step Header/Toolbar -->
                    <div
                      class="bg-surface-container-low px-6 py-3 flex justify-between items-center border-b border-outline-variant/30"
                    >
                      <div class="flex items-center gap-3">
                        <div
                          class="w-6 h-6 rounded-full bg-primary text-on-primary flex items-center justify-center text-xs font-bold shadow-sm"
                        >
                          {{ stepIndex + 1 }}
                        </div>
                        <span class="text-sm font-bold opacity-80">
                          {{
                            getStepGroup(stepIndex)?.get('name')?.value ||
                              getStepGroup(stepIndex)?.get('id')?.value ||
                              '未命名步骤'
                          }}
                        </span>
                      </div>
                      <div class="flex items-center gap-1">
                        @if (isHandset()) {
                          <button
                            mat-icon-button
                            icon-button-center
                            [disabled]="stepIndex === 0"
                            (click)="moveStep(stepIndex, -1)"
                            matTooltip="上移"
                            type="button"
                            class="!w-8 !h-8"
                          >
                            <mat-icon class="!text-lg">arrow_upward</mat-icon>
                          </button>
                          <button
                            mat-icon-button
                            icon-button-center
                            [disabled]="stepIndex === steps.length - 1"
                            (click)="moveStep(stepIndex, 1)"
                            matTooltip="下移"
                            type="button"
                            class="!w-8 !h-8"
                          >
                            <mat-icon class="!text-lg">arrow_downward</mat-icon>
                          </button>
                        }
                        <button
                          mat-icon-button
                          icon-button-center
                          color="warn"
                          (click)="removeStep(stepIndex)"
                          matTooltip="删除此步骤"
                          class="!w-8 !h-8"
                        >
                          <mat-icon class="!text-lg">delete_outline</mat-icon>
                        </button>
                      </div>
                    </div>

                    <div [formGroup]="getStepGroup(stepIndex)!" class="p-6 flex flex-col gap-4">
                      <!-- Execution Condition at the Top -->
                      <mat-form-field appearance="outline" class="w-full">
                        <mat-label>执行条件 (If)</mat-label>
                        <input
                          matInput
                          formControlName="if"
                          placeholder="例如：steps.step1.outputs.code == '200'"
                        />
                        <mat-hint>Go-Expr 表达式，支持变量引用</mat-hint>
                      </mat-form-field>

                      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        <mat-form-field appearance="outline">
                          <mat-label>步骤 ID</mat-label>
                          <input matInput formControlName="id" placeholder="例如：fetch_data" />
                          <mat-hint
                            >用于 <code>{{ '\${{ steps.ID.outputs.key }}' }}</code> 引用</mat-hint
                          >
                          @if (getStepGroup(stepIndex)?.get('id')?.errors?.['pattern']) {
                            <mat-error>仅限小写字母、数字、下划线</mat-error>
                          }
                        </mat-form-field>
                        <mat-form-field appearance="outline">
                          <mat-label>显示名称 (可选)</mat-label>
                          <input matInput formControlName="name" placeholder="默认为步骤 ID" />
                        </mat-form-field>
                      </div>

                      <div class="flex flex-col gap-1.5">
                        <mat-label class="text-[10px] font-bold text-outline uppercase px-1"
                          >处理器类型</mat-label
                        >

                        <div
                          (click)="openProcessorSelector(stepIndex)"
                          class="group relative flex items-center gap-4 p-4 rounded-2xl border border-outline-variant/30 hover:border-primary/50 hover:bg-primary/[0.02] cursor-pointer transition-all duration-200"
                          [class.bg-primary/5]="getStepManifest(stepIndex)"
                          [class.border-primary/20]="getStepManifest(stepIndex)"
                        >
                          <div
                            class="w-12 h-12 rounded-xl bg-surface-container-high flex items-center justify-center shrink-0 border border-outline-variant/20 group-hover:bg-primary/10 transition-colors"
                          >
                            <mat-icon
                              class="!w-6 !h-6 !text-[24px] text-outline group-hover:text-primary transition-colors"
                              >extension</mat-icon
                            >
                          </div>

                          <div class="flex-1 min-w-0">
                            @if (getStepManifest(stepIndex); as manifest) {
                              <div class="flex items-baseline gap-2">
                                <span class="text-sm font-bold text-primary">{{
                                  manifest.name
                                }}</span>
                                <span class="text-[10px] font-mono text-outline opacity-50"
                                  >标识: {{ manifest.id }}</span
                                >
                              </div>
                              <p class="text-xs text-outline opacity-80 mt-0.5 line-clamp-1">
                                {{ manifest.description }}
                              </p>
                            } @else {
                              <span class="text-sm text-outline opacity-50 font-medium"
                                >点击选择一个任务处理器...</span
                              >
                            }
                          </div>

                          <mat-icon
                            class="text-outline opacity-30 group-hover:opacity-100 group-hover:text-primary transition-all"
                            >chevron_right</mat-icon
                          >
                        </div>

                        @if (
                          getStepGroup(stepIndex)?.get('type')?.errors?.['required'] &&
                          getStepGroup(stepIndex)?.get('type')?.touched
                        ) {
                          <p class="text-[10px] text-error px-1 mt-1">必须选择一个处理器</p>
                        }
                      </div>

                      <!-- Dynamic Parameters -->
                      @if (
                        getProcessorParams(getStepGroup(stepIndex)?.get('type')?.value);
                        as params
                      ) {
                        @if (params.length > 0) {
                          <div class="mt-2 space-y-4">
                            <p
                              class="text-[10px] font-bold uppercase tracking-wider text-outline px-1"
                            >
                              输入参数配置
                            </p>
                            <div
                              formGroupName="params"
                              class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4"
                            >
                              @for (param of params; track param.name) {
                                <div class="flex flex-col">
                                  <app-discovery-suggest-input
                                    [label]="
                                      (param.description || param.name) +
                                      (param.optional ? ' (可选)' : '')
                                    "
                                    [formControlName]="param.name!"
                                    [code]="param.lookupCode || ''"
                                    [staticSuggestions]="getOutputReferences(stepIndex)"
                                    [hint]="getParamHint(param)"
                                  ></app-discovery-suggest-input>
                                  @if (
                                    getStepGroup(stepIndex)?.get('params')?.get(param.name!)
                                      ?.errors?.['regexMatch']
                                  ) {
                                    <p class="text-[10px] text-error px-1 mt-1">
                                      值不符合该参数的前端正则要求
                                    </p>
                                  }
                                  @if (
                                    getStepGroup(stepIndex)?.get('params')?.get(param.name!)
                                      ?.errors?.['required'] &&
                                    getStepGroup(stepIndex)?.get('params')?.get(param.name!)
                                      ?.touched
                                  ) {
                                    <p class="text-[10px] text-error px-1 mt-1">此参数必填</p>
                                  }
                                </div>
                              }
                            </div>
                          </div>
                        }
                      }
                    </div>
                  </div>
                }

                <div class="flex justify-center py-2">
                  <button
                    mat-stroked-button
                    color="primary"
                    (click)="addStep()"
                    type="button"
                    class="!px-8 border-dashed border-2 inline-flex items-center gap-2"
                  >
                    <mat-icon class="!m-0">add</mat-icon>
                    <span>添加一个新的步骤</span>
                  </button>
                </div>

                <div class="mt-6 flex gap-2">
                  <button mat-button matStepperPrevious type="button">上一步</button>
                  <button mat-flat-button matStepperNext type="button" color="primary">
                    预览配置
                  </button>
                </div>
              </div>
            </mat-step>

            <!-- Step 4: Review -->
            <mat-step>
              <ng-template matStepLabel>确认保存</ng-template>
              <div
                class="mt-8 p-8 bg-surface border border-outline-variant rounded-2xl text-center max-w-2xl mx-auto"
              >
                <mat-icon class="text-5xl h-auto w-auto text-primary opacity-20 mb-4"
                  >verified</mat-icon
                >
                <h3 class="text-xl font-bold mb-2">准备就绪</h3>
                <p class="text-on-surface-variant mb-8 text-sm">
                  工作流 <strong>{{ infoForm.value.name }}</strong> 已配置完成，包含
                  {{ steps.length }} 个任务步骤。 点击顶部的“保存”按钮即可完成操作。
                </p>
                <div class="flex justify-center gap-4">
                  <button mat-button matStepperPrevious type="button">返回修改</button>
                  <button
                    mat-flat-button
                    color="primary"
                    (click)="submit()"
                    [disabled]="!infoForm.valid || steps.length === 0"
                  >
                    立即保存
                  </button>
                </div>
              </div>
            </mat-step>
          </mat-stepper>
        </div>
      </div>
    </div>
  `,
  styles: [
    `
      :host {
        display: block;
        height: 100vh;
      }
      ::ng-deep .mat-stepper-vertical {
        background: transparent !important;
      }
      ::ng-deep .mat-step-header {
        border-radius: 12px !important;
        margin-bottom: 8px !important;
      }
    `,
  ],
})
export class CreateWorkflowDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private orchService = inject(ActionsService);
  private rbacService = inject(RbacService);
  private cdr = inject(ChangeDetectorRef);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  manifests = signal<ModelsStepManifest[]>([]);
  serviceAccounts = signal<ModelsServiceAccount[]>([]);

  manifestMap = computed(() => {
    const map = new Map<string, ModelsStepManifest>();
    this.manifests().forEach((m) => {
      if (m.id) map.set(m.id, m);
    });
    return map;
  });

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
  ) {}

  async ngOnInit() {
    try {
      const manifests = await firstValueFrom(this.orchService.actionsManifestsGet());
      this.manifests.set(manifests || []);
    } catch (e) {
      console.error('Failed to load initial data', e);
    }

    requestAnimationFrame(() => {
      if (this.data.workflow) {
        this.infoForm.patchValue({
          name: this.data.workflow.name,
          description: this.data.workflow.description,
          enabled: this.data.workflow.enabled ?? true,
          timeout: this.data.workflow.timeout ?? 7200,
          serviceAccountId: this.data.workflow.serviceAccountId,
          cronEnabled: this.data.workflow.cronEnabled,
          cronExpr: this.data.workflow.cronExpr,
          webhookEnabled: this.data.workflow.webhookEnabled,
        });

        if (this.data.workflow.vars) {
          Object.entries(this.data.workflow.vars).forEach(([key, def]) => {
            this.addVar(key, def);
          });
        }

        if (this.data.workflow.steps) {
          for (const s of this.data.workflow.steps) {
            this.addStep(s);
          }
        }
      } else {
        this.addStep(); // Start with one empty step
      }
      this.cdr.detectChanges();
    });
  }

  getVarGroup(index: number): FormGroup {
    return this.vars.at(index) as FormGroup;
  }

  addVar(key?: string, def?: any) {
    const varGroup = this.fb.group({
      key: [key || '', [Validators.required, Validators.pattern('^[a-z0-9_]+$')]],
      description: [def?.description || ''],
      default: [def?.default || ''],
      required: [def?.required || false],
      regexFrontend: [def?.regexFrontend || ''],
      regexBackend: [def?.regexBackend || ''],
    });

    // Apply regex validator to 'default' field
    varGroup.get('default')?.setValidators([this.createRegexValidator(varGroup, 'regexFrontend')]);

    this.vars.push(varGroup);
  }

  createRegexValidator(group: FormGroup, regexFieldName: string): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
      const value = control.value;
      if (!value) return null;

      // Skip validation if it contains ${{ ... }}
      if (value.includes('${{')) return null;

      const regexStr = group.get(regexFieldName)?.value;
      if (!regexStr) return null;

      try {
        const regex = new RegExp(regexStr);
        if (!regex.test(value)) {
          return { regexMatch: true };
        }
      } catch (e) {
        return null;
      }
      return null;
    };
  }

  removeVar(index: number) {
    this.vars.removeAt(index);
  }

  hasRegex(index: number): boolean {
    const group = this.getVarGroup(index);
    return !!(group.get('regexFrontend')?.value || group.get('regexBackend')?.value);
  }

  openVarExtra(index: number) {
    const group = this.getVarGroup(index);
    const dialogRef = this.dialog.open(VariableConfigDialogComponent, {
      width: '500px',
      data: {
        regexFrontend: group.get('regexFrontend')?.value,
        regexBackend: group.get('regexBackend')?.value,
      },
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        group.patchValue({
          regexFrontend: result.regexFrontend,
          regexBackend: result.regexBackend,
        });
        this.cdr.markForCheck();
      }
    });
  }

  getStepGroup(index: number): FormGroup | undefined {
    if (index < 0 || index >= this.steps.length) return undefined;
    return this.steps.at(index) as FormGroup;
  }

  addStep(stepData?: ModelsStep) {
    const defaultID = `step_${this.steps.length + 1}`;
    const stepGroup = this.fb.group({
      _uid: [Math.random().toString(36).substring(2)], // Unique ID for tracking
      id: [stepData?.id || defaultID, [Validators.required, Validators.pattern('^[a-z0-9_]+$')]],
      name: [stepData?.name || ''], // Optional
      if: [stepData?.if || ''], // Conditional execution
      type: [stepData?.type || '', Validators.required],
      params: this.fb.group({}),
    });

    this.steps.push(stepGroup);

    if (stepData) {
      this.onProcessorChange(this.steps.length - 1, stepData.params);
    }
  }

  removeStep(index: number) {
    this.steps.removeAt(index);
  }

  moveStep(index: number, delta: number) {
    const to = index + delta;
    if (to < 0 || to >= this.steps.length) return;

    const control = this.steps.at(index);
    this.steps.removeAt(index);
    this.steps.insert(to, control);
    this.cdr.markForCheck();
  }

  drop(event: CdkDragDrop<any[]>) {
    const from = event.previousIndex;
    const to = event.currentIndex;
    if (from === to) return;

    const control = this.steps.at(from);
    this.steps.removeAt(from);
    this.steps.insert(to, control);

    this.cdr.markForCheck();
  }

  openProcessorSelector(index: number) {
    const stepGroup = this.getStepGroup(index);
    if (!stepGroup) return;

    const dialogRef = this.dialog.open(ProcessorSelectorDialogComponent, {
      width: '600px',
      data: {
        manifests: this.manifests(),
        selectedId: stepGroup.get('type')?.value,
      },
    });

    dialogRef.afterClosed().subscribe((selectedId) => {
      if (selectedId) {
        stepGroup.get('type')?.setValue(selectedId);
        this.onProcessorChange(index);
        this.cdr.markForCheck();
      }
    });
  }

  getStepManifest(index: number): ModelsStepManifest | undefined {
    const type = this.getStepGroup(index)?.get('type')?.value;
    if (!type) return undefined;
    return this.manifestMap().get(type);
  }

  onProcessorChange(index: number, initialParams?: any) {
    const stepGroup = this.getStepGroup(index);
    if (!stepGroup) return;

    const type = stepGroup.get('type')?.value;
    const manifest = this.manifestMap().get(type);

    const paramsGroup = stepGroup.get('params') as FormGroup;
    // Clear existing params
    Object.keys(paramsGroup.controls).forEach((key) => paramsGroup.removeControl(key));

    if (manifest && manifest.params) {
      for (const p of manifest.params) {
        if (p.name) {
          const validators: ValidatorFn[] = p.optional ? [] : [Validators.required];
          if (p.regexFrontend) {
            validators.push((control: AbstractControl): ValidationErrors | null => {
              const val = control.value;
              if (!val || val.includes('${{')) return null;
              try {
                const regex = new RegExp(p.regexFrontend!);
                if (!regex.test(val)) return { regexMatch: true };
              } catch (e) {
                return null;
              }
              return null;
            });
          }
          paramsGroup.addControl(
            p.name,
            this.fb.control(initialParams?.[p.name] || '', validators),
          );
        }
      }
    }
  }

  getProcessorParams(type: string | undefined): ModelsParamDefinition[] {
    if (!type) return [];
    const manifest = this.manifestMap().get(type);
    if (!manifest) return [];
    return manifest.params || [];
  }

  getParamHint(param: ModelsParamDefinition): string {
    if (param.regexFrontend) {
      return `格式要求: ${param.regexFrontend}`;
    }
    return '';
  }

  getOutputReferences(currentIndex: number): string[] {
    const refs: string[] = [];

    // Add global vars
    for (const v of this.vars.value) {
      if (v.key) {
        refs.push(`\${{ vars.${v.key} }}`);
      }
    }

    for (let i = 0; i < currentIndex; i++) {
      const step = this.getStepGroup(i);
      if (!step) continue;

      const stepID = step.get('id')?.value;
      const type = step.get('type')?.value;
      const manifest = this.manifestMap().get(type);

      if (stepID && manifest && manifest.outputParams) {
        for (const op of manifest.outputParams) {
          if (op.name) {
            refs.push(`\${{ steps.${stepID}.outputs.${op.name} }}`);
          }
        }
      }
    }
    return refs;
  }

  async submit() {
    const varsMap: { [key: string]: ModelsVarDefinition } = {};
    for (const v of this.vars.value) {
      if (v.key) {
        varsMap[v.key] = {
          description: v.description,
          default: v.default,
          required: v.required,
          regexFrontend: v.regexFrontend,
          regexBackend: v.regexBackend,
        };
      }
    }

    const workflow: ModelsWorkflow = {
      ...this.infoForm.value,
      vars: varsMap,
      steps: this.steps.value,
    };

    try {
      // Pre-save validation via backend
      await firstValueFrom(this.orchService.actionsWorkflowsValidatePost(workflow));
      this.dialogRef.close(workflow);
    } catch (err: any) {
      const errorMsg = err.error?.message || '配置校验失败';
      this.snackBar.open(errorMsg, '确定', {
        duration: 5000,
      });
      // Keep dialog open if validation fails
    }
  }
}
