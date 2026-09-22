<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
import CodeSnippet from '@/components/help/CodeSnippet.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        All the notifications that Sasso sends go through your email, but sometimes the mail is not
        the best way to receive notifications.
      </p>
      <p>
        For this reason, Sasso allows you to add a Telegram bot to your account and receive
        notifications through it. You have to create a bot, add it to a chat and add both the bot
        token and the chat id to Sasso. After that, you will start receiving notifications on the
        chat as well.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Create a Telegram bot</template>
      <p>
        To create a Telegram bot, you probably have to talk with the BotFather on Telegram. Follow
        the
        <a href="https://core.telegram.org/bots/tutorial" target="_blank" class="link link-primary">
          official Telegram documentation
        </a>
        to create a bot and get the token.
      </p>
      <p>A bot token is a string that looks like this:</p>
      <CodeSnippet> 1213232159:AFanfvdjrenfdv_MJsjesk345jfdsks </CodeSnippet>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Get the chat id</template>
      <p>
        To get the chat id, the idea is to send a message to the bot in the cat you have created and
        use the Telegram API to get the id of the chat. Search on the internet for a tutorial on how
        to do this.
      </p>
      <p>A chat id is usually a negative number that looks like this:</p>
      <CodeSnippet> -123456789 </CodeSnippet>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Name and Notes</template>
      <p>
        Both name and notes are only for you, so you can write anything you want in them. A good
        idea is to write the name of the chat you have created, so you can easily distinguish
        between multiple bots if you have more than one.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Actions</template>
      <p>
        You can test your bot using the "Test" button. This will send a test notification to the
        chat you have specified.
      </p>
      <p>
        <b>You can have multiple bots</b> and enable/disable them as you like. If you disable a bot,
        you will stop receiving notifications on it.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
