import {
  launchJupyterLab as apiLaunchJupyterLab,
  previewJupyterLab as apiPreviewJupyterLab,
} from 'services/api';
import { RawJson } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { openCommandResponse } from 'utils/wait';

export interface JupyterLabOptions {
  name?: string;
  pool?: string;
  slots?: number;
  template?: string;
  workspaceId?: number;
}

interface JupyterLabLaunchOptions extends JupyterLabOptions {
  config?: RawJson;
}

export const launchJupyterLab = async (options: JupyterLabLaunchOptions = {}): Promise<void> => {
  try {
    const commandResponse = await apiLaunchJupyterLab({
      config: options.config || {
        description: options.name === '' ? undefined : options.name,
        resources: {
          resource_pool: options.pool === '' ? undefined : options.pool,
          slots: options.slots,
        },
      },
      templateName: options.template === '' ? undefined : options.template,
      workspaceId: options.workspaceId,
    });
    openCommandResponse(commandResponse);
  } catch (e) {
    handleError(e, {
      level: ErrorLevel.Error,
      silent: false,
      type: ErrorType.Server,
    });
  }
};

export const previewJupyterLab = async (options: JupyterLabOptions = {}): Promise<RawJson> => {
  try {
    const config = await apiPreviewJupyterLab({
      config: {
        description: options.name === '' ? undefined : options.name,
        resources: {
          resource_pool: options.pool === '' ? undefined : options.pool,
          slots: options.slots,
        },
      },
      preview: true,
      templateName: options.template === '' ? undefined : options.template,
      workspaceId: options.workspaceId,
    });
    return config;
  } catch (e) {
    throw new Error('Unable to load JupyterLab config.');
  }
};
