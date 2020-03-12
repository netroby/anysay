import request from '@/utils/request';
import { TableListParams, TableListItem } from './data.d';

export async function queryRule(params?: TableListParams) {
  return request('/api/listSay', {
    params,
  });
}


export async function addRule(params: TableListItem) {
  return request('/api/addSay', {
    method: 'POST',
    data: {
      ...params,
      method: 'post',
    },
  });
}
