> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimax.io/docs/llms.txt
> Use this file to discover all available pages before exploring further.

<AgentInstructions>

## Submitting Feedback

If you encounter incorrect, outdated, or confusing documentation on this page, submit feedback:

POST https://platform.minimax.io/docs/feedback

```json
{
  "path": "/guides/text-m2-reasoning",
  "feedback": "Description of the issue"
}
```

Only submit feedback when you have something specific and actionable to report.

</AgentInstructions>

# What makes good reasoning data

> MiniMax M2, ranks Top-1 among open-source models and Top-5 among all models

Artificial Analysis is a comprehensive benchmark that reflects the diversity of models’ reasoning abilities. Our newly released model, MiniMax M2, ranks Top-1 among open-source models and Top-5 among all models.

![Artificial Analysis Intelligence Index](https://filecdn.minimax.chat/public/3df653de-629e-48a3-89f0-e44fe52686ed.png)

In the past, community discussions on improving reasoning abilities often focused on optimizing RL algorithms or constructing verifiable data in domains like Math and Code. In the M2 project, we conducted more "general" explorations. As a member of the Reasoning team, I'd like to share some of our findings and thoughts on data — what makes good reasoning data.

## Quality of CoT and Response

The quality of CoT is reflected in its logical completeness without excessive redundancy. For instance, in instruction following tasks, overly brief CoT often leads to models skipping steps or being overconfident, causing significant harm to the model's final performance and capability generalization. For responses, we noticed that most open-source work overfits certain benchmark format patterns to achieve better leaderboard scores. While this is effective for single data directions, it severely hinders capability generalization for a general-purpose model. Therefore, when synthesizing data, we introduced format diversity and observed significant gains in multi-directional fusion experiments. Meanwhile, for potential bad cases in CoT and responses, such as hallucinations, instruction-following failures, and logical errors. We performed data cleaning using rules + LLM-as-a-judge. By continuously iterating on this misalignment elimination pipeline, we've become increasingly convinced that every bad case has its corresponding dirty training data, and improvements in data quality will inevitably be reflected in model performance.

## Difficulty and Diversity of Data Distribution

Like many discussions in the community, our experiments also found that math and code data are critical for improving reasoning capabilities. The reasoning abilities brought by these two types of data often benefit all tasks, such as STEM and IF. However, we also found that we still need sufficiently diverse data to cover more domains, such as logical reasoning, science, instruction following, and open-ended creative tasks. Tasks from different domains have different thinking paradigms, and the diversity of reasoning is the foundation for capability generalization. Additionally, we noticed in our experiments that harder and more complex queries are more effective for model training, so we adjusted data distribution based on pass rate (for verifiable tasks) or complexity scores (for non-verifiable tasks).

## Data Scaling

Finally, an old but important topic: Scaling. When data quality and diversity meet the standards, increasing data scale consistently brings significant gains. Whether it's increasing the number of queries, doing 1Q-multiple-A, multi-epoch training, or even mixing data from different directions to bring more training steps, the model steadily improves. In practice, data scaling is a highly engineering-oriented problem, so we attempted to consolidate all data based on task characteristics, dividing them into two data pipelines: Verifiable and Non-Verifiable, for automated data synthesis and processing. In fact, the Reasoning team is almost entirely composed of interns, and this data pipeline effectively ensured team collaboration efficiency and consistency in data output.

## Future Work

Moving forward, we will continue to delve deeper in two directions. One is compound capabilities, such as knowledge + reasoning, and the enhancement of reasoning tasks by tools in Agent scenarios. The other is how to integrate Verifiable and Non-Verifiable tasks, such as the fusion of CoT across different domains and the generalization of reasoning capabilities, as well as the unification of training methods. Our team is also continuously progressing and growing. We welcome interested colleagues to join the discussion. Happy to chat!
