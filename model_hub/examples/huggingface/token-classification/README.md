# Token Classification
This example mirrors the [token-classification example](https://github.com/huggingface/transformers/tree/master/examples/token-classification) from the original huggingface transformers repo.  

## Files
* **ner_trial.py**: The core code implementing the [Determined trial interface](https://docs.determined.ai/latest/reference/api/pytorch.html#pytorch-trial) for this example.  This is super easy with the [helper functions provided in model-hub](../model_hub/transformers/_trial.py).
* **ner_utils.py**: Utility functions for NER largely extracted from [run_ner.py](https://github.com/huggingface/transformers/tree/master/examples/token-classification/run_ner.py) to separate example code from determined code.
* **startup-hook.sh**: Installs additional dependencies for this example matching those in [requirements.txt](https://github.com/huggingface/transformers/tree/master/examples/token-classification/requirements.txt) of the source example.

### Configuration Files
* **ner_config.yaml**: Configuration for finetuning on the CoNLL-2003 dataset with BERT.  These values match the [default values](https://github.com/huggingface/transformers/blob/master/src/transformers/training_args.py) used for transformer's Trainer.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Make sure the environment variable `DET_MASTER` is set to your cluster URL.
Then you run the following command from the command line: `det experiment create -f ner_config.yaml .`. 

### Configuration
To run with your own data, change the following fields in `ner_config.yaml`:
* `dataset_name: null`
* `train_file: <path_to_train_file>`
* `validation_file: <path_to_validation_file>`

To run with multiple GPUs (whether same node or multiple nodes), change `slots_per_trial` to the desired
degree of parallelism.  You will likely want to change `global_batch_size` so that each GPU will
process `global_batch_size / slots_per_trial` batches per iteration and adjust the `learning_rate`
to be compatible with a larger or smaller batch size.  

